package framework

import (
	"context"

	"github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/karmada-io/karmada/pkg/util/fedinformer"
	"github.com/karmada-io/karmada/pkg/util/gclient"
	"github.com/karmada-io/karmada/pkg/util/restmapper"
)

// InitControllerManagerAndStartCache initializes the controller manager and starts the cache.
func InitControllerManagerAndStartCache(restConfig *rest.Config) ctrl.Manager {
	manager, err := ctrl.NewManager(restConfig, ctrl.Options{
		Logger:         klog.Background(),
		Scheme:         gclient.NewSchema(),
		MapperProvider: restmapper.MapperProvider,
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			opts.DefaultTransform = fedinformer.StripUnusedFields
			return cache.New(config, opts)
		},
	})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	go func() {
		_ = manager.GetCache().Start(context.TODO())
	}()
	synced := manager.GetCache().WaitForCacheSync(context.TODO())
	gomega.Expect(synced).Should(gomega.BeTrue())

	return manager
}
