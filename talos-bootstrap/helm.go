package main

import (
	"context"
	"time"

	"github.com/mittwald/go-helm-client"
	"github.com/mittwald/go-helm-client/values"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

func setupHelmClient(logger *UILogger) (helmclient.Client, error) {
	helmClientOptions := &helmclient.Options{
		Output: NewUILoggerWriter(logger),
		Debug:  true, // Enable debug logging for Helm operations
		DebugLog: func(format string, v ...interface{}) {
			logger.Infof(format, v...)
		},
		Linting: true, // Enable chart linting,
	}

	kubeclientOptions := helmclient.KubeConfClientOptions{
		Options:    helmClientOptions,
		KubeConfig: kubeconfig, //kubeconfig from prev. step
	}

	return helmclient.NewClientFromKubeConf(&kubeclientOptions)
}

func helmInstallArgo(helmClient helmclient.Client) (*release.Release, error) {
	err := helmClient.AddOrUpdateChartRepo(repo.Entry{
		Name: "argo",
		URL:  "https://argoproj.github.io/argo-helm",
	})
	if err != nil {
		return nil, err
	}

	chartSpec := helmclient.ChartSpec{
		ReleaseName: "stolos-argocd",
		Description: "ArgoCD Deployed by Stolos Cloud bootstrapper",
		ChartName:   "argo/argo-cd",
		Namespace:   "stolos-argocd",
		ValuesOptions: values.Options{
			ValueFiles: []string{
				"./argo.default.values.yaml",
				//"../k8s-manifests/argocd/helm/values.yaml"
			},
		},
		Version:         "8.5.2",
		CreateNamespace: true,
		DisableHooks:    false,
		Wait:            true,
		UpgradeCRDs:     true,
		Timeout:         2 * time.Minute,
	}

	return helmClient.InstallChart(context.Background(), &chartSpec, &helmclient.GenericHelmOptions{})
}
