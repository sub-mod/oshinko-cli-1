/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"fmt"
	"net/http"

	"github.com/kubernetes-incubator/service-catalog/pkg/api"
	genericapiserverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage/etcd3/preflight"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/service-catalog/pkg/apiserver"
	"github.com/kubernetes-incubator/service-catalog/pkg/apiserver/options"
	"github.com/kubernetes-incubator/service-catalog/pkg/registry/servicecatalog/server"
)

// RunServer runs an API server with configuration according to opts
func RunServer(opts *ServiceCatalogServerOptions, stopCh <-chan struct{}) error {
	storageType, err := opts.StorageType()
	if err != nil {
		return err
	}
	if stopCh == nil {
		/* the caller of RunServer should generate the stop channel
		if there is a need to stop the API server */
		stopCh = make(chan struct{})
	}

	err = opts.Validate()
	if nil != err {
		return err
	}

	if storageType == server.StorageTypeEtcd {
		return runEtcdServer(opts, stopCh)
	}
	// This should never happen, catch for potential bugs
	panic("Unexpected storage type: " + storageType)
}

func runEtcdServer(opts *ServiceCatalogServerOptions, stopCh <-chan struct{}) error {
	etcdOpts := opts.EtcdOptions
	glog.V(4).Infoln("Preparing to run API server")
	genericConfig, scConfig, err := buildGenericConfig(opts)
	if err != nil {
		return err
	}

	glog.V(4).Infoln("Creating storage factory")

	// The API server stores objects using a particular API version for each
	// group, regardless of API version of the object when it was created.
	//
	// storageGroupsToEncodingVersion holds a map of API group to version that
	// the API server uses to store that group.
	storageGroupsToEncodingVersion, err := options.NewStorageSerializationOptions().StorageGroupsToEncodingVersion()
	if err != nil {
		return fmt.Errorf("error generating storage version map: %s", err)
	}

	// Build the default storage factory.
	//
	// The default storage factory returns the storage interface for a
	// particular GroupResource (an (api-group, resource) tuple).
	storageFactory, err := apiserver.NewStorageFactory(
		etcdOpts.StorageConfig,
		etcdOpts.DefaultStorageMediaType,
		api.Codecs,
		genericapiserverstorage.NewDefaultResourceEncodingConfig(api.Registry),
		storageGroupsToEncodingVersion,
		nil, /* group storage version overrides */
		apiserver.DefaultAPIResourceConfigSource(),
		nil, /* resource config overrides */
	)
	if err != nil {
		glog.Errorf("error creating storage factory: %v", err)
		return err
	}

	// // Set the finalized generic and storage configs
	config := apiserver.NewEtcdConfig(genericConfig, 0 /* deleteCollectionWorkers */, storageFactory)

	// Fill in defaults not already set in the config
	completed := config.Complete()

	// make the server
	glog.V(4).Infoln("Completing API server configuration")
	server, err := completed.NewServer()
	if err != nil {
		return fmt.Errorf("error completing API server configuration: %v", err)
	}
	addPostStartHooks(server.GenericAPIServer, scConfig, stopCh)

	// Install healthz checks before calling PrepareRun.
	etcdChecker := checkEtcdConnectable{
		ServerList: etcdOpts.StorageConfig.ServerList,
	}
	// PingHealtz is installed by the default config, so it will
	// run in addition the checkers being installed here.
	server.GenericAPIServer.AddHealthzChecks(etcdChecker)

	// do we need to do any post api installation setup? We should have set up the api already?
	glog.Infoln("Running the API server")
	server.PrepareRun().Run(stopCh)

	return nil
}

// checkEtcdConnectable is a HealthzChecker that makes sure the
// etcd storage backend is up and contactable.
type checkEtcdConnectable struct {
	ServerList []string
}

func (c checkEtcdConnectable) Name() string {
	return "etcd"
}

func (c checkEtcdConnectable) Check(_ *http.Request) error {
	glog.Info("etcd checker called")
	serverReachable, err := preflight.EtcdConnection{ServerList: c.ServerList}.CheckEtcdServers()

	if err != nil {
		glog.Errorf("etcd checker failed with err: %v", err)
		return err
	}
	if !serverReachable {
		msg := "etcd failed to reach any server"
		glog.Error(msg)
		return fmt.Errorf(msg)
	}
	return nil
}
