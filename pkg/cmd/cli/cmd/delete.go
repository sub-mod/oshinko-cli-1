package cmd

import (
	"fmt"
	"io"
	//"reflect"
	"strings"
	//"github.com/renstrom/dedent"
	"github.com/spf13/cobra"
	//"k8s.io/kubernetes/pkg/api/meta"
	//"k8s.io/kubernetes/pkg/kubectl"

	//"k8s.io/kubernetes/pkg/kubectl/resource"
	//"k8s.io/kubernetes/pkg/runtime"
	utilerrors "k8s.io/kubernetes/pkg/util/errors"
	//"k8s.io/kubernetes/pkg/watch"

	//"k8s.io/kubernetes/pkg/api"

	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	//cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	//DELETE
	"github.com/openshift/origin/pkg/client"
	cliconfig "github.com/openshift/origin/pkg/cmd/cli/config"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"time"
)

func NewCmdDelete(fullName string, f *clientcmd.Factory, out io.Writer) *cobra.Command {
	cmd := CmdDelete(f, out)
	cmd.Long = fmt.Sprintf(getLong, fullName)
	cmd.Example = fmt.Sprintf(getExample, fullName)
	cmd.SuggestFor = []string{"list"}
	return cmd
}

func CmdDelete(f *clientcmd.Factory, out io.Writer) *cobra.Command {
	options := &ClusterOptions{}

	// retrieve a list of handled resources from printer as valid args
	validArgs := []string{}
	//p, err := f.Printer(nil, kubectl.PrintOptions{
	//	ColumnLabels: []string{},
	//})
	//cmdutil.CheckErr(err)
	//if p != nil {
	//	validArgs = p.HandledResources()
	//	argAliases = kubectl.ResourceAliases(validArgs)
	//}

	cmd := &cobra.Command{
		Use:     "delete ([-f FILENAME] | TYPE [(NAME | -l label | --all)])",
		Short:   "Delete resources by filenames, stdin, resources and names, or by resources and label selector.",
		Long:    getLong,
		Example: getExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.PathOptions = cliconfig.NewPathOptions(cmd)

			if err := options.Complete(f, args, out); err != nil {
				kcmdutil.CheckErr(kcmdutil.UsageError(cmd, err.Error()))
			}
			if err := options.RunDelete(out, cmd, args); err != nil {
				kcmdutil.CheckErr(err)
			}
		},
		SuggestFor: []string{"list", "ps"},
		ValidArgs:  validArgs,
		//ArgAliases: argAliases,
	}
	//[SUBIN]removing this
	//cmdutil.AddPrinterFlags(cmd)
	//cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml|wide|name|custom-columns=...|custom-columns-file=...|go-template=...|go-template-file=...|jsonpath=...|jsonpath-file=... See custom columns [http://kubernetes.io/docs/user-guide/kubectl-overview/#custom-columns], golang template [http://golang.org/pkg/text/template/#pkg-overview] and jsonpath template [http://kubernetes.io/docs/user-guide/jsonpath].")
	//cmd.Flags().Bool("all-namespaces", false, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
	//cmd.Flags().Bool("export", false, "If true, use 'export' for the resources.  Exported resources are stripped of cluster-specific information.")

	//delete
	cmd.Flags().BoolVarP(&options.DisplayShort, "short", "q", false, "If true, display only the cluster names")
	return cmd
}

// RunGet implements the generic Get command
// TODO: convert all direct flag accessors to a struct and pass that instead of cmd
func (o ClusterOptions) RunDelete(out io.Writer, cmd *cobra.Command, args []string) error {
	allErrs := []error{}

	config := o.Config
	//clientCfg := o.ClientConfig
	//out := o.Out

	currentContext := config.Contexts[config.CurrentContext]
	currentProject := currentContext.Namespace

	kclient := o.KClient
	oclient := o.Client

	//fmt.Println("Deletion : ", args, currentProject)
	info := deleteCluster(args[1], currentProject, oclient, kclient)
	if info != "" {
		fmt.Println("Deletion may be incomplete:")
		//return reterr(fail(nil, "Deletion may be incomplete: "+info, 500))
	}

	if _, err := fmt.Fprintf(out, "cluster \"%s\" deleted \n",
		args[1],
	); err != nil {
		allErrs = append(allErrs, err)
	}
	return utilerrors.NewAggregate(allErrs)
}

func waitForCount(client kclient.ReplicationControllerInterface, name string, count int32) {

	for i := 0; i < 5; i++ {
		r, _ := client.Get(name)
		if int32(r.Status.Replicas) == count {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func deleteCluster(clustername, namespace string, osclient *client.Client, client *kclient.Client) string {

	info := []string{}
	scalerepls := []string{}

	// Build a selector list for the "oshinko-cluster" label
	selectorlist := makeSelector("", clustername)

	// Delete all of the deployment configs
	dcc := osclient.DeploymentConfigs(namespace)
	deployments, err := dcc.List(selectorlist)
	if err != nil {
		info = append(info, "unable to find deployment configs ("+err.Error()+")")
	}
	for i := range deployments.Items {
		name := deployments.Items[i].Name
		err = dcc.Delete(name)
		if err != nil {
			info = append(info, "unable to delete deployment config "+name+" ("+err.Error()+")")
		}
	}

	// Get a list of all the replication controllers for the cluster
	// and set all of the replica values to 0
	rcc := client.ReplicationControllers(namespace)
	repls, err := rcc.List(selectorlist)
	if err != nil {
		info = append(info, "unable to find replication controllers ("+err.Error()+")")
	}
	for i := range repls.Items {
		name := repls.Items[i].Name
		repls.Items[i].Spec.Replicas = 0
		_, err = rcc.Update(&repls.Items[i])
		if err != nil {
			info = append(info, "unable to scale replication controller "+name+" ("+err.Error()+")")
		} else {
			scalerepls = append(scalerepls, name)
		}
	}

	// Wait for the replica count to drop to 0 for each one we scaled
	for i := range scalerepls {
		waitForCount(rcc, scalerepls[i], 0)
	}

	// Delete each replication controller
	for i := range repls.Items {
		name := repls.Items[i].Name
		err = rcc.Delete(name)
		if err != nil {
			info = append(info, "unable to delete replication controller "+name+" ("+err.Error()+")")
		}
	}

	// Delete the services
	sc := client.Services(namespace)
	srvs, err := sc.List(selectorlist)
	if err != nil {
		info = append(info, "unable to find services ("+err.Error()+")")
	}
	for i := range srvs.Items {
		name := srvs.Items[i].Name
		err = sc.Delete(name)
		if err != nil {
			info = append(info, "unable to delete service "+name+" ("+err.Error()+")")
		}
	}
	return strings.Join(info, ", ")
}
