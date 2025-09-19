// configserver.go
package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/container"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
)

var configBundle *bundle.Bundle

// Machines stores the Machines we have already seen in IP-Hostname pairs

type SaveState struct {
	ClusterEndpoint string        `json:"ClusterEndpoint"`
	BootstrapInfo   BootstrapInfo `json:"BootstrapInfo"`
	MachinesCache   Machines      `json:"MachinesCache"`
}

type Machines struct {
	ControlPlanes map[string][]byte `json:"ControlPlanes"` // map IP : Hostname
	Workers       map[string][]byte `json:"Workers"`       // map IP : Hostname
}

var saveState SaveState

// StartConfigServer starts a minimal HTTP server with /machineconfig.
func StartConfigServer(logger *UILogger, addr string) error {

	if !doRestoreProgress {
		machines := Machines{
			ControlPlanes: make(map[string][]byte),
			Workers:       make(map[string][]byte),
		}
		saveState = SaveState{
			BootstrapInfo: *bootstrapInfos,
			MachinesCache: machines,
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/machineconfig", machineConfigHandler(logger))
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Successf("Config server listening on %s", addr)
	return srv.ListenAndServe()
}

// machineConfigHandler handles GET /machineconfig?h=${hostname}&m=${mac}&s=${serial}&u=${uuid}
func machineConfigHandler(logger *UILogger) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		//host := q.Get("h")
		mac := q.Get("m")
		//serial := q.Get("s")
		uuid := q.Get("u")
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		responseWriter.Header().Set("Content-Type", "application/yaml")

		_, isSeenControlPlane := saveState.MachinesCache.ControlPlanes[uuid]
		isFirstMachine := len(saveState.MachinesCache.ControlPlanes) == 0

		if isFirstMachine || isSeenControlPlane {
			configBytes, err := handleControlPlane(logger, ip, mac, uuid)
			if err != nil {
				logger.Errorf("Error handling control plane request: %v", err)
				responseWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, err = responseWriter.Write(configBytes)
			if err != nil {
				logger.Errorf("ErError writing response: %v", err)
				responseWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			saveState.MachinesCache.ControlPlanes[uuid] = configBytes
			saveState.ClusterEndpoint = fmt.Sprintf("https://%s:6443", ip)
			saveStateToJSON(logger)
		} else {
			configBytes, err := handleWorker(logger, ip, mac, uuid)
			if err != nil {
				logger.Errorf("Error handling control plane request: %v", err)
				responseWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, err = responseWriter.Write(configBytes)
			if err != nil {
				logger.Errorf("Error writing response: %v", err)
				responseWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			saveState.MachinesCache.Workers[uuid] = configBytes
			saveStateToJSON(logger)
		}
	}
}

func handleControlPlane(logger *UILogger, ip string, mac string, uuid string) ([]byte, error) {
	var err error
	logger.Infof("HTTP Request from %s ! Generating controlplane machineconfig on the fly...", ip)

	cachedConfig, alreadyPresent := saveState.MachinesCache.ControlPlanes[uuid]
	if alreadyPresent {
		logger.Infof("ControlPlane with IP (%s) already seen! Re-sending config...", ip)
		return cachedConfig, nil
	}

	if configBundle == nil {
		configBundle, err = CreateMachineConfigBundle(ip)
		if err != nil {
			logger.Errorf("Error generating talosconfig: %v", err)
			logger.Errorf(err.Error())
		}
	}

	cfg := &v1alpha1.Config{
		ConfigVersion: "v1alpha1",
		MachineConfig: &v1alpha1.MachineConfig{
			MachineNetwork: &v1alpha1.NetworkConfig{
				NetworkHostname: "controlplane-0",
			},
		},
	}

	ctr := container.NewV1Alpha1(cfg)
	patch := configpatcher.NewStrategicMergePatch(ctr)
	err = configBundle.ApplyPatches([]configpatcher.Patch{patch}, true, false)

	steps[3].IsDone = true
	return configBundle.Serialize(encoder.CommentsDocs, machine.TypeControlPlane)
}

func handleWorker(logger *UILogger, ip string, mac string, uuid string) ([]byte, error) {
	cachedConfig, alreadyPresent := saveState.MachinesCache.Workers[uuid]
	if alreadyPresent {
		logger.Infof("Worker with IP (%s) and hostname (%s) already seen! Re-sending config...", ip)
		return cachedConfig, nil
	}

	logger.Infof("Found Worker %s , Responded with worker.machineconfig.yaml", ip)

	cfg := &v1alpha1.Config{
		ConfigVersion: "v1alpha1",
		MachineConfig: &v1alpha1.MachineConfig{
			MachineNetwork: &v1alpha1.NetworkConfig{
				NetworkHostname: fmt.Sprintf("worker-%d", len(saveState.MachinesCache.Workers)),
			},
		},
	}
	ctr := container.NewV1Alpha1(cfg)
	patch := configpatcher.NewStrategicMergePatch(ctr)
	_ = configBundle.ApplyPatches([]configpatcher.Patch{patch}, false, true)

	return configBundle.Serialize(encoder.CommentsDocs, machine.TypeWorker)
}
