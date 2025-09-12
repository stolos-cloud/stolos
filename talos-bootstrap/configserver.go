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
	return srv.ListenAndServe()
}

// machineConfigHandler handles GET /machineconfig?h=${hostname}&m=${mac}&s=${serial}&u=${uuid}
func machineConfigHandler(logger *UILogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		host := q.Get("h")
		mac := q.Get("m")
		serial := q.Get("s")
		uuid := q.Get("u")
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		// Example log lines (control plane on first hit, workers afterwards)
		if atomic.CompareAndSwapInt32(&firstHit, 0, 1) {
			logger.Infof("HTTP Request from %s ! Generating controlplane machineconfig on the fly...", ip)
			logger.Successf("Saved to %s....", "./controlplane.machineconfig.yaml")
			logger.Infof("Responding to HTTP request with controlplane machineconfig ...")
			logger.Infof("Generating talosconfig with endpoint https://%s:6443 on the fly...", ip)
			logger.Successf("Talosconfig saved to %s", "./talosconfig")
		} else {
			logger.Infof("Found Worker %s , Responded with worker.machineconfig.yaml", ip)
		}

		// TODO: Build and return the actual machineconfig bytes here.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "ok",
			"message":  "machineconfig served (scaffold)",
			"hostname": host,
			"mac":      mac,
			"serial":   serial,
			"uuid":     uuid,
		})
	}
}
