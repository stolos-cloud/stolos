package types

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type Stolos struct {
	metav1.TypeMeta
	metav1.ObjectMeta `json:"metadata"`
	Spec              StolosSpec `json:"spec"`
}

type StolosSpec struct {
	ClusterName          string               `json:"clusterName"`
	BaseDomain           string               `json:"baseDomain"`
	LocalPathProvisioner LocalPathProvisioner `json:"localPathProvisioner"`
	MetalLB              MetalLB              `json:"metallb"`
	ArgoCD               ArgoCD               `json:"argocd"`
	Contour              Contour              `json:"contour"`
	CertManager          CertManager          `json:"certManager"`
	CNPG                 CNPG                 `json:"cnpg"`
	StolosPlatform       StolosPlatform       `json:"stolosPlatform"`
}

type LocalPathProvisioner struct {
	Deploy    bool   `json:"deploy" Default:"false"`
	Namespace string `json:"namespace" Default:"\"local-path-storage\""`
	Version   string `json:"version" Default:"\"v0.0.32\""`
}

type MetalLB struct {
	Deploy       bool   `json:"deploy" Default:"true"`
	ConfigureArp bool   `json:"configureArp" Default:"true"`
	ArpIp        string `json:"arpIp"`
	Namespace    string `json:"namespace" Default:"\"metallb-system\""`
	Version      string `json:"version"`
}

type ArgoCD struct {
	Deploy              bool   `json:"deploy" Default:"true"`
	Namespace           string `json:"namespace" Default:"\"argocd\""`
	Version             string `json:"version"`
	Subdomain           string `json:"subdomain"`
	ImageUpdaterVersion string `json:"imageUpdaterVersion"`
	RepositoryOwner     string `json:"repositoryOwner"`
	RepositoryName      string `json:"repositoryName"`
	RepositoryRevision  string `json:"repositoryRevision" Default:"\"main\""`
}

type Contour struct {
	Deploy    bool   `json:"deploy" Default:"true"`
	Namespace string `json:"namespace" Default:"\"projectcontour\""`
	Version   string `json:"version"`
}

type CertManager struct {
	Deploy               bool   `json:"deploy" Default:"true"`
	Version              string `json:"version"`
	Namespace            string `json:"namespace" Default:"\"cert-manager\""`
	ClusterIssuerProd    string `json:"clusterIssuerProd" Default:"\"letsencrypt-prod\""`
	ClusterIssuerStaging string `json:"clusterIssuerStaging" Default:"\"letsencrypt-staging\""`
	DefaultClusterIssuer string `json:"defaultClusterIssuer" Default:"\"letsencrypt-prod\""`
	SelfSigned           bool   `json:"selfSigned" Default:"false"`
	Email                string `json:"email"`
}

type CNPG struct {
	Deploy        bool   `json:"deploy" Default:"true"`
	Namespace     string `json:"namespace" Default:"\"cnpg-system\""`
	Version       string `json:"version"`
	BarmanVersion string `json:"barmanVersion"`
}

type StolosPlatform struct {
	Deploy               bool         `json:"deploy" Default:"true"`
	Namespace            string       `json:"namespace" Default:"\"stolos-system\""`
	BackendSubdomain     string       `json:"backendSubdomain"`
	FrontendSubdomain    string       `json:"frontendSubdomain"`
	Database             CnpgDbConfig `json:"database"`
	DefaultAdminPassword string       `json:"defaultAdminPassword"`
	DefaultAdminEmail    string       `json:"defaultAdminEmail"`
	PathToYaml           string       `json:"pathToYaml"`
}

type CnpgDbConfig struct {
	Image           string `json:"image"`
	InstanceCount   int    `json:"instanceCount" Default:"1"`
	SizeInGigabytes int    `json:"sizeInGigabytes" Default:"5"`
}
