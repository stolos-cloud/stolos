package main

import (
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/gcp"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/github"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
)

type BootstrapInfo struct {
	TalosInfo  talos.TalosInfo   `json:"TalosInfo" field_required:"true"`
	GCPInfo    gcp.GCPConfig     `json:"GCPInfo" field_required:"false"`
	GitHubInfo github.GitHubInfo `json:"GitHubInfo" field_required:"false"`
}

type SaveState struct {
	ClusterEndpoint        string                          `json:"ClusterEndpoint"`
	BootstrapInfo          BootstrapInfo                   `json:"BootstrapInfo"`
	MachinesCache          talos.Machines                  `json:"MachinesCache"`
	MachinesDisks          talos.MachinesDisks             `json:"MachinesDisks"`
	GitHubApp              github.AppManifest              `json:"GitHubApp"`
	GitHubAppInstallResult github.AppInstallCallbackResult `json:"GitHubAppInstallResult"`
}

var ConfigBundle *bundle.Bundle
