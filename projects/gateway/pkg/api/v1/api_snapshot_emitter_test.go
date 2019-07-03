// Code generated by solo-kit. DO NOT EDIT.

// +build solokit

package v1

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/kubeutils"
	"github.com/solo-io/go-utils/log"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	kuberc "github.com/solo-io/solo-kit/pkg/api/v1/clients/kube"
	"github.com/solo-io/solo-kit/test/helpers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// Needed to run tests in GKE
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	// From https://github.com/kubernetes/client-go/blob/53c7adfd0294caa142d961e1f780f74081d5b15f/examples/out-of-cluster-client-configuration/main.go#L31
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var _ = Describe("V1Emitter", func() {
	if os.Getenv("RUN_KUBE_TESTS") != "1" {
		log.Printf("This test creates kubernetes resources and is disabled by default. To enable, set RUN_KUBE_TESTS=1 in your env.")
		return
	}
	var (
		namespace1           string
		namespace2           string
		name1, name2         = "angela" + helpers.RandString(3), "bob" + helpers.RandString(3)
		cfg                  *rest.Config
		kube                 kubernetes.Interface
		emitter              ApiEmitter
		virtualServiceClient VirtualServiceClient
		gatewayClient        GatewayClient
	)

	BeforeEach(func() {
		namespace1 = helpers.RandString(8)
		namespace2 = helpers.RandString(8)
		kube = helpers.MustKubeClient()
		err := kubeutils.CreateNamespacesInParallel(kube, namespace1, namespace2)
		Expect(err).NotTo(HaveOccurred())
		cfg, err = kubeutils.GetConfig("", "")
		Expect(err).NotTo(HaveOccurred())
		// VirtualService Constructor
		virtualServiceClientFactory := &factory.KubeResourceClientFactory{
			Crd:         VirtualServiceCrd,
			Cfg:         cfg,
			SharedCache: kuberc.NewKubeCache(context.TODO()),
		}

		virtualServiceClient, err = NewVirtualServiceClient(virtualServiceClientFactory)
		Expect(err).NotTo(HaveOccurred())
		// Gateway Constructor
		gatewayClientFactory := &factory.KubeResourceClientFactory{
			Crd:         GatewayCrd,
			Cfg:         cfg,
			SharedCache: kuberc.NewKubeCache(context.TODO()),
		}

		gatewayClient, err = NewGatewayClient(gatewayClientFactory)
		Expect(err).NotTo(HaveOccurred())
		emitter = NewApiEmitter(virtualServiceClient, gatewayClient)
	})
	AfterEach(func() {
		err := kubeutils.DeleteNamespacesInParallelBlocking(kube, namespace1, namespace2)
		Expect(err).NotTo(HaveOccurred())
	})
	It("tracks snapshots on changes to any resource", func() {
		ctx := context.Background()
		err := emitter.Register()
		Expect(err).NotTo(HaveOccurred())

		snapshots, errs, err := emitter.Snapshots([]string{namespace1, namespace2}, clients.WatchOpts{
			Ctx:         ctx,
			RefreshRate: time.Second,
		})
		Expect(err).NotTo(HaveOccurred())

		var snap *ApiSnapshot

		/*
			VirtualService
		*/

		assertSnapshotVirtualServices := func(expectVirtualServices VirtualServiceList, unexpectVirtualServices VirtualServiceList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectVirtualServices {
						if _, err := snap.VirtualServices.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectVirtualServices {
						if _, err := snap.VirtualServices.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := virtualServiceClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := virtualServiceClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		virtualService1a, err := virtualServiceClient.Write(NewVirtualService(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		virtualService1b, err := virtualServiceClient.Write(NewVirtualService(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotVirtualServices(VirtualServiceList{virtualService1a, virtualService1b}, nil)
		virtualService2a, err := virtualServiceClient.Write(NewVirtualService(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		virtualService2b, err := virtualServiceClient.Write(NewVirtualService(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotVirtualServices(VirtualServiceList{virtualService1a, virtualService1b, virtualService2a, virtualService2b}, nil)

		err = virtualServiceClient.Delete(virtualService2a.GetMetadata().Namespace, virtualService2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = virtualServiceClient.Delete(virtualService2b.GetMetadata().Namespace, virtualService2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotVirtualServices(VirtualServiceList{virtualService1a, virtualService1b}, VirtualServiceList{virtualService2a, virtualService2b})

		err = virtualServiceClient.Delete(virtualService1a.GetMetadata().Namespace, virtualService1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = virtualServiceClient.Delete(virtualService1b.GetMetadata().Namespace, virtualService1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotVirtualServices(nil, VirtualServiceList{virtualService1a, virtualService1b, virtualService2a, virtualService2b})

		/*
			Gateway
		*/

		assertSnapshotGateways := func(expectGateways GatewayList, unexpectGateways GatewayList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectGateways {
						if _, err := snap.Gateways.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectGateways {
						if _, err := snap.Gateways.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := gatewayClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := gatewayClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		gateway1a, err := gatewayClient.Write(NewGateway(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		gateway1b, err := gatewayClient.Write(NewGateway(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotGateways(GatewayList{gateway1a, gateway1b}, nil)
		gateway2a, err := gatewayClient.Write(NewGateway(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		gateway2b, err := gatewayClient.Write(NewGateway(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotGateways(GatewayList{gateway1a, gateway1b, gateway2a, gateway2b}, nil)

		err = gatewayClient.Delete(gateway2a.GetMetadata().Namespace, gateway2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = gatewayClient.Delete(gateway2b.GetMetadata().Namespace, gateway2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotGateways(GatewayList{gateway1a, gateway1b}, GatewayList{gateway2a, gateway2b})

		err = gatewayClient.Delete(gateway1a.GetMetadata().Namespace, gateway1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = gatewayClient.Delete(gateway1b.GetMetadata().Namespace, gateway1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotGateways(nil, GatewayList{gateway1a, gateway1b, gateway2a, gateway2b})
	})
	It("tracks snapshots on changes to any resource using AllNamespace", func() {
		ctx := context.Background()
		err := emitter.Register()
		Expect(err).NotTo(HaveOccurred())

		snapshots, errs, err := emitter.Snapshots([]string{""}, clients.WatchOpts{
			Ctx:         ctx,
			RefreshRate: time.Second,
		})
		Expect(err).NotTo(HaveOccurred())

		var snap *ApiSnapshot

		/*
			VirtualService
		*/

		assertSnapshotVirtualServices := func(expectVirtualServices VirtualServiceList, unexpectVirtualServices VirtualServiceList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectVirtualServices {
						if _, err := snap.VirtualServices.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectVirtualServices {
						if _, err := snap.VirtualServices.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := virtualServiceClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := virtualServiceClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		virtualService1a, err := virtualServiceClient.Write(NewVirtualService(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		virtualService1b, err := virtualServiceClient.Write(NewVirtualService(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotVirtualServices(VirtualServiceList{virtualService1a, virtualService1b}, nil)
		virtualService2a, err := virtualServiceClient.Write(NewVirtualService(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		virtualService2b, err := virtualServiceClient.Write(NewVirtualService(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotVirtualServices(VirtualServiceList{virtualService1a, virtualService1b, virtualService2a, virtualService2b}, nil)

		err = virtualServiceClient.Delete(virtualService2a.GetMetadata().Namespace, virtualService2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = virtualServiceClient.Delete(virtualService2b.GetMetadata().Namespace, virtualService2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotVirtualServices(VirtualServiceList{virtualService1a, virtualService1b}, VirtualServiceList{virtualService2a, virtualService2b})

		err = virtualServiceClient.Delete(virtualService1a.GetMetadata().Namespace, virtualService1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = virtualServiceClient.Delete(virtualService1b.GetMetadata().Namespace, virtualService1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotVirtualServices(nil, VirtualServiceList{virtualService1a, virtualService1b, virtualService2a, virtualService2b})

		/*
			Gateway
		*/

		assertSnapshotGateways := func(expectGateways GatewayList, unexpectGateways GatewayList) {
		drain:
			for {
				select {
				case snap = <-snapshots:
					for _, expected := range expectGateways {
						if _, err := snap.Gateways.Find(expected.GetMetadata().Ref().Strings()); err != nil {
							continue drain
						}
					}
					for _, unexpected := range unexpectGateways {
						if _, err := snap.Gateways.Find(unexpected.GetMetadata().Ref().Strings()); err == nil {
							continue drain
						}
					}
					break drain
				case err := <-errs:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(time.Second * 10):
					nsList1, _ := gatewayClient.List(namespace1, clients.ListOpts{})
					nsList2, _ := gatewayClient.List(namespace2, clients.ListOpts{})
					combined := append(nsList1, nsList2...)
					Fail("expected final snapshot before 10 seconds. expected " + log.Sprintf("%v", combined))
				}
			}
		}
		gateway1a, err := gatewayClient.Write(NewGateway(namespace1, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		gateway1b, err := gatewayClient.Write(NewGateway(namespace2, name1), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotGateways(GatewayList{gateway1a, gateway1b}, nil)
		gateway2a, err := gatewayClient.Write(NewGateway(namespace1, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		gateway2b, err := gatewayClient.Write(NewGateway(namespace2, name2), clients.WriteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotGateways(GatewayList{gateway1a, gateway1b, gateway2a, gateway2b}, nil)

		err = gatewayClient.Delete(gateway2a.GetMetadata().Namespace, gateway2a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = gatewayClient.Delete(gateway2b.GetMetadata().Namespace, gateway2b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotGateways(GatewayList{gateway1a, gateway1b}, GatewayList{gateway2a, gateway2b})

		err = gatewayClient.Delete(gateway1a.GetMetadata().Namespace, gateway1a.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())
		err = gatewayClient.Delete(gateway1b.GetMetadata().Namespace, gateway1b.GetMetadata().Name, clients.DeleteOpts{Ctx: ctx})
		Expect(err).NotTo(HaveOccurred())

		assertSnapshotGateways(nil, GatewayList{gateway1a, gateway1b, gateway2a, gateway2b})
	})
})
