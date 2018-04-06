/*
Copyright 2018 The Kubernetes Authors.

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

// This file was automatically generated by informer-gen

package v1

import (
	time "time"

	versioned "github.com/solo-io/gloo/pkg/storage/crd/client/clientset/versioned"
	internalinterfaces "github.com/solo-io/gloo/pkg/storage/crd/client/informers/externalversions/internalinterfaces"
	v1 "github.com/solo-io/gloo/pkg/storage/crd/client/listers/solo.io/v1"
	solo_io_v1 "github.com/solo-io/gloo/pkg/storage/crd/solo.io/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// VirtualHostInformer provides access to a shared informer and lister for
// VirtualHosts.
type VirtualHostInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.VirtualHostLister
}

type virtualHostInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewVirtualHostInformer constructs a new informer for VirtualHost type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewVirtualHostInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredVirtualHostInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredVirtualHostInformer constructs a new informer for VirtualHost type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredVirtualHostInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.GlooV1().VirtualHosts(namespace).List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.GlooV1().VirtualHosts(namespace).Watch(options)
			},
		},
		&solo_io_v1.VirtualHost{},
		resyncPeriod,
		indexers,
	)
}

func (f *virtualHostInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredVirtualHostInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *virtualHostInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&solo_io_v1.VirtualHost{}, f.defaultInformer)
}

func (f *virtualHostInformer) Lister() v1.VirtualHostLister {
	return v1.NewVirtualHostLister(f.Informer().GetIndexer())
}
