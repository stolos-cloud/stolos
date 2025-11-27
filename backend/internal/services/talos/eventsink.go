package talos

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/siderolabs/siderolink/pkg/events"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	"gorm.io/gorm"
)

// starts the Talos event sink gRPC server to receive events from booting nodes
func (s *TalosService) StartEventSink() {
	// Skip if event sink hostname is not configured
	if s.cfg.Talos.EventSinkHostname == "" {
		log.Println("Talos event sink hostname not configured, skipping event sink startup")
		return
	}

	log.Printf("Starting Talos event sink on %s:%s", s.cfg.Talos.EventSinkBindHostname, s.cfg.Talos.EventSinkPort)

	// Prepare talosInfo struct for EventSink (use bind hostname for actual binding)
	talosInfo := &talos.TalosInfo{
		HTTPHostname: s.cfg.Talos.EventSinkBindHostname,
		HTTPPort:     s.cfg.Talos.EventSinkPort,
	}

	// Start EventSink
	go func() {
		err := talos.EventSink(talosInfo, func(ctx context.Context, event events.Event) error {
			// Extract IP from event.Node
			ip := strings.Split(event.Node, ":")[0]

			if slices.Contains(ignoredNodesCache, ip) {
				// Skip ignored node.
				return nil
			}

			// Check if node already exists
			var existing models.Node
			err := s.db.Where("ip_address = ? AND provider = ?", ip, "onprem").First(&existing).Error

			if err == gorm.ErrRecordNotFound {

				// Try to connect, add to ignoredNodeCache if fails.

				cli, err := GetInsecureMachineryClient(ctx, ip)

				if err != nil {
					// Failing to create client, but don't ignore the node yet.
					return errors.Wrapf(err, "error creating insecure machinery client")
				}

				status, err := GetMachineStatus(cli)

				if err != nil {
					//ignoredNodesCache = append(ignoredNodesCache, ip)
					log.Printf("Ignoring node %s: %v", ip, err)
					return errors.Wrapf(err, "Error connecting to node %s, skipping", ip)
				}

				var mac string
				if iface := GetMachineBestExternalNetworkInterface(ctx, cli); iface != nil {
					mac = iface.Mac
				}

				log.Printf("Found machine at %s with stage %s", ip, status.Stage.String())

				var cluster models.Cluster
				s.db.First(&cluster, models.Cluster{})

				// Auto-register new node in the DB as Pending.
				node := models.Node{
					Name:         fmt.Sprintf("node-%s", strings.ReplaceAll(ip, ".", "-")),
					IPAddress:    ip,
					Provider:     "onprem",
					Status:       models.StatusPending,
					MACAddress:   mac,
					Architecture: "Unknown",
					ClusterID:    cluster.ID,
				}

				if err := s.db.Create(&node).Error; err != nil {
					log.Printf("Failed to auto-register node %s: %v", ip, err)
					return err
				}

				log.Printf("Auto-registered new on-prem node: %s (IP: %s)", node.Name, ip)

				if s.wsManager != nil {
					s.wsManager.BroadcastToSessionType(wsservices.SessionTypeEvent, wsservices.Message{
						Type: "NewPendingNodeDetected",
						Payload: map[string]any{
							"node": node,
						},
					})
				}
			} else if err != nil {
				log.Printf("Error checking node existence for IP %s: %v", ip, err)
				return err
			} else {
				log.Printf("Node with IP %s already registered", ip)
			}

			return nil
		})

		if err != nil {
			log.Printf("Talos event sink error: %v", err)
		}
	}()

	log.Printf("Talos event sink started successfully")
}
