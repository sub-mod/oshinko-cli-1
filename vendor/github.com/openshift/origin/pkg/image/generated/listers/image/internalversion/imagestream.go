// This file was automatically generated by lister-gen

package internalversion

import (
	image "github.com/openshift/origin/pkg/image/apis/image"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ImageStreamLister helps list ImageStreams.
type ImageStreamLister interface {
	// List lists all ImageStreams in the indexer.
	List(selector labels.Selector) (ret []*image.ImageStream, err error)
	// ImageStreams returns an object that can list and get ImageStreams.
	ImageStreams(namespace string) ImageStreamNamespaceLister
	ImageStreamListerExpansion
}

// imageStreamLister implements the ImageStreamLister interface.
type imageStreamLister struct {
	indexer cache.Indexer
}

// NewImageStreamLister returns a new ImageStreamLister.
func NewImageStreamLister(indexer cache.Indexer) ImageStreamLister {
	return &imageStreamLister{indexer: indexer}
}

// List lists all ImageStreams in the indexer.
func (s *imageStreamLister) List(selector labels.Selector) (ret []*image.ImageStream, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*image.ImageStream))
	})
	return ret, err
}

// ImageStreams returns an object that can list and get ImageStreams.
func (s *imageStreamLister) ImageStreams(namespace string) ImageStreamNamespaceLister {
	return imageStreamNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ImageStreamNamespaceLister helps list and get ImageStreams.
type ImageStreamNamespaceLister interface {
	// List lists all ImageStreams in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*image.ImageStream, err error)
	// Get retrieves the ImageStream from the indexer for a given namespace and name.
	Get(name string) (*image.ImageStream, error)
	ImageStreamNamespaceListerExpansion
}

// imageStreamNamespaceLister implements the ImageStreamNamespaceLister
// interface.
type imageStreamNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ImageStreams in the indexer for a given namespace.
func (s imageStreamNamespaceLister) List(selector labels.Selector) (ret []*image.ImageStream, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*image.ImageStream))
	})
	return ret, err
}

// Get retrieves the ImageStream from the indexer for a given namespace and name.
func (s imageStreamNamespaceLister) Get(name string) (*image.ImageStream, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(image.Resource("imagestream"), name)
	}
	return obj.(*image.ImageStream), nil
}
