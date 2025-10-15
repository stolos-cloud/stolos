package types

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type Stolos struct {
	metav1.TypeMeta
	metav1.ObjectMeta `json:"metadata"`
	Spec              StolosSpec `json:"spec"`
}

type StolosSpec struct {
	ClusterName string      `json:"clusterName"`
	BaseDomain  string      `json:"baseDomain"`
	MetalLB     MetalLB     `json:"metallb"`
	ArgoCD      ArgoCD      `json:"argocd"`
	Contour     Contour     `json:"contour"`
	CertManager CertManager `json:"certManager"`
	CNPG        CNPG        `json:"cnpg"`
}

type MetalLB struct {
	Deploy       bool   `json:"deploy"`
	ConfigureArp bool   `json:"configureArp"`
	ArpIp        string `json:"arpIp"`
	Namespace    string `json:"namespace"`
	Version      string `json:"version"`
}

type ArgoCD struct {
	Deploy              bool   `json:"deploy"`
	Namespace           string `json:"namespace"`
	Version             string `json:"version"`
	Subdomain           string `json:"subdomain"`
	ImageUpdaterVersion string `json:"imageUpdaterVersion"`
}

type Contour struct {
	Deploy    bool   `json:"deploy"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

type CertManager struct {
	Deploy               bool   `json:"deploy"`
	Version              string `json:"version"`
	Namespace            string `json:"namespace"`
	ClusterIssuerProd    string `json:"clusterIssuerProd"`
	ClusterIssuerStaging string `json:"clusterIssuerStaging"`
	DefaultClusterIssuer string `json:"defaultClusterIssuer"`
	SelfSigned           bool   `json:"selfSigned"`
	Email                string `json:"email"`
}

type CNPG struct {
	Deploy        bool   `json:"deploy"`
	Namespace     string `json:"namespace"`
	Version       string `json:"version"`
	BarmanVersion string `json:"barmanVersion"`
}
