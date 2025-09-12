// configserver.go
package main

import (
	"encoding/json"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

var firstHit int32 // atomically track first control-plane hit

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
			handleControlPlane(logger, responseWriter, ip, mac, host, serial, uuid)
		} else {
			handleWorker(logger, responseWriter, ip, mac, host, serial, uuid)
		}
	}
}

func handleControlPlane(logger *UILogger, w http.ResponseWriter, ip string, mac string, host string, serial string, uuid string) {
	logger.Infof("HTTP Request from %s ! Generating controlplane machineconfig on the fly...", ip)
	logger.Successf("Saved to %s....", "./controlplane.machineconfig.yaml")
	logger.Infof("Responding to HTTP request with controlplane machineconfig ...")
	logger.Infof("Generating talosconfig with endpoint https://%s:6443 on the fly...", ip)
	logger.Successf("Talosconfig saved to %s", "./talosconfig")

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"message":  "CONTROL PLANE",
		"hostname": host,
		"mac":      mac,
		"serial":   serial,
		"uuid":     uuid,
	})
}

func handleWorker(logger *UILogger, w http.ResponseWriter, ip string, mac string, host string, serial string, uuid string) {
	logger.Infof("Found Worker %s , Responded with worker.machineconfig.yaml", ip)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"message":  "WORKER",
		"hostname": host,
		"mac":      mac,
		"serial":   serial,
		"uuid":     uuid,
	})
}
