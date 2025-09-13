// configserver.go
package main

import (
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/container"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
)

var firstHit int32 // atomically track first control-plane hit
var configBundle *bundle.Bundle
var workersCount = 0

// StartConfigServer starts a minimal HTTP server with /machineconfig.
func StartConfigServer(logger *UILogger, addr string) error {
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
		var err error
		q := r.URL.Query()
		host := q.Get("h")
		mac := q.Get("m")
		serial := q.Get("s")
		uuid := q.Get("u")
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		responseWriter.Header().Set("Content-Type", "application/json")

		// Example log lines (control plane on first hit, workers afterwards)
		if atomic.CompareAndSwapInt32(&firstHit, 0, 1) {
			// FIRST HIT - CONTROL PLANE
			err = handleControlPlane(logger, responseWriter, ip, mac, host, serial, uuid)
		} else {
			err = handleWorker(logger, responseWriter, ip, mac, host, serial, uuid)
		}

		if err != nil {
			logger.Errorf("Error handling control plane request: %v", err)
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func handleControlPlane(logger *UILogger, w http.ResponseWriter, ip string, mac string, host string, serial string, uuid string) error {
	var err error
	logger.Infof("HTTP Request from %s ! Generating controlplane machineconfig on the fly...", ip)
	logger.Successf("Saved to %s....", "./controlplane.machineconfig.yaml")
	logger.Infof("Responding to HTTP request with controlplane machineconfig ...")
	logger.Infof("Generating talosconfig with endpoint https://%s:6443 on the fly...", ip)
	logger.Successf("Talosconfig saved to %s", "./talosconfig")

	if configBundle == nil {
		configBundle, err = CreateMachineConfigBundle(ip)
		if err != nil {
			logger.Errorf("Error generating talosconfig: %v", err)
			panic(err)
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

	configBytes, err := configBundle.Serialize(encoder.CommentsDocs, machine.TypeControlPlane)
	_, err = w.Write(configBytes)
	if err != nil {
		return err
	}

	return nil
}

func handleWorker(logger *UILogger, w http.ResponseWriter, ip string, mac string, host string, serial string, uuid string) error {
	var err error
	logger.Infof("Found Worker %s , Responded with worker.machineconfig.yaml", ip)

	cfg := &v1alpha1.Config{
		ConfigVersion: "v1alpha1",
		MachineConfig: &v1alpha1.MachineConfig{
			MachineNetwork: &v1alpha1.NetworkConfig{
				NetworkHostname: fmt.Sprintf("worker-%d", workersCount),
			},
		},
	}
	ctr := container.NewV1Alpha1(cfg)
	patch := configpatcher.NewStrategicMergePatch(ctr)
	err = configBundle.ApplyPatches([]configpatcher.Patch{patch}, false, true)

	configBytes, err := configBundle.Serialize(encoder.CommentsDocs, machine.TypeWorker)
	_, err = w.Write(configBytes)
	if err != nil {
		return err
	}

	workersCount++
	return nil
}
