package helm

import (
	"context"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/go-helm-client/values"
	"github.com/stolos-cloud/stolos-bootstrap/internal/logging"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

func SetupHelmClient(logger *logging.Logger, kubeconfig []byte) (helmclient.Client, error) {
	helmClientOptions := &helmclient.Options{
		Debug: true, // Enable debug logging for Helm operations
		DebugLog: func(format string, v ...interface{}) {
			(*logger).Infof(format, v...)
		},
		Linting: true, // Enable chart linting,
	}

	kubeclientOptions := helmclient.KubeConfClientOptions{
		Options:    helmClientOptions,
		KubeConfig: kubeconfig, //kubeconfig from prev. step
	}

	return helmclient.NewClientFromKubeConf(&kubeclientOptions)
}

func HelmInstallArgo(helmClient helmclient.Client, releaseName string, namespace string, valuesFiles []string) (*release.Release, error) {
	err := helmClient.AddOrUpdateChartRepo(repo.Entry{
		Name: "argo",
		URL:  "https://argoproj.github.io/argo-helm",
	})
	if err != nil {
		return nil, err
	}

	chartSpec := helmclient.ChartSpec{
		ReleaseName: "argocd",
		Description: "ArgoCD Deployed by Stolos Cloud",
		ChartName:   "argo/argo-cd",
		Namespace:   namespace,
		ValuesOptions: values.Options{
			ValueFiles: valuesFiles,
		},
		Version:         "8.5.2",
		CreateNamespace: true,
		DisableHooks:    false,
		Wait:            true,
		UpgradeCRDs:     true,
		Timeout:         2 * time.Minute,
	}

	return helmClient.InstallOrUpgradeChart(context.Background(), &chartSpec, &helmclient.GenericHelmOptions{})
}
