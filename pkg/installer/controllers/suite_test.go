/*
Copyright 2022.

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

package controllers

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"kubegems.io/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment

	ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx = logf.IntoContext(ctx, ctrl.Log)

	By("bootstrapping test environment")

	home, _ := os.UserHomeDir()
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{"../../../deploy/charts/kubegems-installer/crds"},
		CRDInstallOptions: envtest.CRDInstallOptions{
			CleanUpAfterUse: true,
		},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join(home, ".local/share/kubebuilder-envtest/k8s/1.20.2-linux-amd64"),
	}
	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = pluginsv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// setup ctrl manager
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:           scheme.Scheme,
		LeaderElectionID: plugins.GroupName,
	})
	Expect(err).NotTo(HaveOccurred())

	// register controller
	err = NewAndSetupPluginReconciler(ctx, mgr, &PluginOptions{})
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).ToNot(HaveOccurred())
	}()
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel() // stop mgr
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Basic Plugin tests", func() {
	It("create remote git helm plugin", func() {
		plugin := &pluginsv1beta1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "local-path-provisioner",
				Namespace: "default",
			},
			Spec: pluginsv1beta1.PluginSpec{
				Kind:    pluginsv1beta1.PluginKindHelm,
				Enabled: true,
				Repo:    "https://github.com/rancher/local-path-provisioner.git",
				Path:    "deploy/chart",
				Version: "v0.0.21", // tag or branch
			},
		}
		err := k8sClient.Create(ctx, plugin)
		Expect(err).NotTo(HaveOccurred())

		waitPhaseSet(ctx, plugin)

		Expect(plugin.Status.Phase).To(Equal(pluginsv1beta1.PluginPhaseInstalled))
		Expect(plugin.Finalizers).To(Equal([]string{PluginFinalizerName}))
		Expect(plugin.Status.Version).To(Equal("0.0.21"))
	})

	It("creates a local helm plugin", func() {
		plugin := &pluginsv1beta1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helm-plugin",
				Namespace: "default",
			},
			Spec: pluginsv1beta1.PluginSpec{
				Kind:    pluginsv1beta1.PluginKindHelm,
				Path:    "testdata/helm-test",
				Enabled: true,
			},
		}
		err := k8sClient.Create(ctx, plugin)
		Expect(err).NotTo(HaveOccurred())

		waitPhaseSet(ctx, plugin)

		Expect(plugin.Status.Phase).To(Equal(pluginsv1beta1.PluginPhaseInstalled))
	})

	It("creates a local native plugin", func() {
		plugin := &pluginsv1beta1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "native-test",
				Namespace: "default",
			},
			Spec: pluginsv1beta1.PluginSpec{
				Kind:    pluginsv1beta1.PluginKindNative,
				Repo:    "file://testdata",
				Path:    "native-test",
				Enabled: true,
				Values: MarshalValues(map[string]interface{}{
					"foo": "barvalue",
				}),
			},
		}
		err := k8sClient.Create(ctx, plugin)
		Expect(err).NotTo(HaveOccurred())

		waitPhaseSet(ctx, plugin)

		dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "native-test", Namespace: "default"}}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(dep), dep)
		Expect(err).NotTo(HaveOccurred())
		Expect(dep.ObjectMeta.Annotations).To(HaveKeyWithValue("foo", "barvalue"))
		Expect(plugin.Status.Phase).To(Equal(pluginsv1beta1.PluginPhaseInstalled))
	})

	It("wait all plugins removed", func() {
		plugins := &pluginsv1beta1.PluginList{}
		err := k8sClient.List(ctx, plugins)
		Expect(err).NotTo(HaveOccurred())
		for _, plugin := range plugins.Items {
			_ = k8sClient.Delete(ctx, &plugin)
		}
		err = waitAllRemoved(ctx)
		Expect(err).NotTo(HaveOccurred())
	})
})

func waitPhaseSet(ctx context.Context, reconciledPlugin *pluginsv1beta1.Plugin) error {
	return wait.PollUntil(time.Second, func() (done bool, err error) {
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(reconciledPlugin), reconciledPlugin); err != nil {
			return false, err
		}
		if reconciledPlugin.Status.Phase == "" {
			return false, nil
		}
		return true, nil
	}, ctx.Done())
}

func waitAllRemoved(ctx context.Context) error {
	return wait.PollUntil(time.Second, func() (done bool, err error) {
		plugins := &pluginsv1beta1.PluginList{}
		if err := k8sClient.List(ctx, plugins, client.InNamespace("default")); err != nil {
			return false, err
		}
		if len(plugins.Items) == 0 {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}
