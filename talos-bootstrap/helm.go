package main

import (
	"context"

	"github.com/mittwald/go-helm-client"
	"github.com/mittwald/go-helm-client/values"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

func setupHelmClient(logger *UILogger) (helmclient.Client, error) {
	helmClientOptions := &helmclient.Options{
		RepositoryConfig: "",
		Debug:            false, // Enable debug logging for Helm operations
		Linting:          true,  // Enable chart linting,
		Output:           NewUILoggerWriter(logger),
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
	}

	return helmClient.InstallChart(context.Background(), &chartSpec, &helmclient.GenericHelmOptions{})
}
