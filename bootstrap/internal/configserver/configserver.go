// configserver.go
package configserver

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/container"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
	"github.com/stolos-cloud/stolos-bootstrap/internal/tui"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/marshal"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/state"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
)

// Machines stores the Machines we have already seen in IP-Hostname pairs

// StartConfigServer starts a minimal HTTP server with /machineconfig.
func StartConfigServer(model *tui.Model, addr string, doRestoreProgress bool, saveState *state.SaveState, bootstrapInfos *state.BootstrapInfo) error {

	if !doRestoreProgress {
		machines := state.Machines{
			ControlPlanes: make(map[string][]byte),
			Workers:       make(map[string][]byte),
		}
		saveState = &state.SaveState{
			BootstrapInfo: *bootstrapInfos,
			MachinesCache: machines,
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/machineconfig", machineConfigHandler(model, saveState, bootstrapInfos))
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	model.Logger.Successf("Config server listening on %s", addr)
	return srv.ListenAndServe()
}

// machineConfigHandler handles GET /machineconfig?h=${hostname}&m=${mac}&s=${serial}&u=${uuid}
func machineConfigHandler(model *tui.Model, saveState *state.SaveState, bootstrapInfos *state.BootstrapInfo) http.HandlerFunc {
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
			configBytes, err := handleControlPlane(model, ip, mac, uuid, *saveState, *bootstrapInfos)
			if err != nil {
				model.Logger.Errorf("Error handling control plane request: %v", err)
				responseWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, err = responseWriter.Write(configBytes)
			if err != nil {
				model.Logger.Errorf("Error writing response: %v", err)
				responseWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			saveState.MachinesCache.ControlPlanes[uuid] = configBytes
			saveState.ClusterEndpoint = fmt.Sprintf("https://%s:6443", ip)
			err = marshal.SaveStateToJSON(*saveState)
			if err != nil {
				model.Logger.Errorf("Error saving state: %v", err)
			}
		} else {
			configBytes, err := handleWorker(model, ip, mac, uuid, *saveState)
			if err != nil {
				model.Logger.Errorf("Error handling control plane request: %v", err)
				responseWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, err = responseWriter.Write(configBytes)
			if err != nil {
				model.Logger.Errorf("Error writing response: %v", err)
				responseWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			saveState.MachinesCache.Workers[uuid] = configBytes
			err = marshal.SaveStateToJSON(*saveState)
			if err != nil {
				model.Logger.Errorf("Error saving state: %v", err)
			}
		}
	}
}

func handleControlPlane(model *tui.Model, ip string, mac string, uuid string, saveState state.SaveState, bootstrapInfos state.BootstrapInfo) ([]byte, error) {
	var err error
	model.Logger.Infof("HTTP Request from %s ! Generating controlplane machineconfig on the fly...", ip)

	cachedConfig, alreadyPresent := saveState.MachinesCache.ControlPlanes[uuid]
	if alreadyPresent {
		model.Logger.Infof("ControlPlane with IP (%s) already seen! Re-sending config...", ip)
		return cachedConfig, nil
	}

	if state.ConfigBundle == nil {
		state.ConfigBundle, err = talos.CreateMachineConfigBundle(ip, bootstrapInfos)
		if err != nil {
			model.Logger.Errorf("Error generating talosconfig: %v", err)
			model.Logger.Errorf(err.Error())
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
	err = state.ConfigBundle.ApplyPatches([]configpatcher.Patch{patch}, true, false)

	tui.SetStepIsDoneByName(model, "WaitControlPlane", true)
	return state.ConfigBundle.Serialize(encoder.CommentsDocs, machine.TypeControlPlane)
}

func handleWorker(model *tui.Model, ip string, mac string, uuid string, saveState state.SaveState) ([]byte, error) {
	cachedConfig, alreadyPresent := saveState.MachinesCache.Workers[uuid]
	if alreadyPresent {
		model.Logger.Infof("Worker with IP (%s) and hostname (%s) already seen! Re-sending config...", ip)
		return cachedConfig, nil
	}

	model.Logger.Infof("Found Worker %s , Responded with worker.machineconfig.yaml", ip)

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
	_ = state.ConfigBundle.ApplyPatches([]configpatcher.Patch{patch}, false, true)

	// TODO : SET IS DONE WHEN 3 WORKER
	// tui.SetStepIsDoneByName(model, "WaitWorker", true)
	return state.ConfigBundle.Serialize(encoder.CommentsDocs, machine.TypeWorker)
}
