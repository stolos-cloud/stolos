package job

import (
	"context"
	"errors"
	"io"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	clusterapi "github.com/siderolabs/talos/pkg/machinery/api/cluster"
	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	clusterres "github.com/siderolabs/talos/pkg/machinery/resources/cluster"
	"github.com/stolos-cloud/stolos/backend/internal/models"
	"github.com/stolos-cloud/stolos/backend/internal/services/node"
	"github.com/stolos-cloud/stolos/backend/internal/services/talos"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// -- job definitions

var ClusterHealthCheckJob = &StolosJob{
	Name:       "ClusterHealthCheckJob",
	Definition: gocron.DurationJob(30 * time.Second),
	JobFunc: func(ts *talos.TalosService, db *gorm.DB) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()

		// --- Get first active node ---
		var node models.Node
		if err := db.
			Model(&models.Node{}).
			Where("status = ?", models.StatusActive).
			First(&node).Error; err != nil {
			log.Printf("no active node found: %v", err)
			return
		}

		if node.IPAddress == "" {
			log.Println("node has no IP address, skipping health check")
			return
		}

		// --- Create Talos machinery client ---
		cli, err := ts.GetMachineryClient(node.IPAddress)
		if err != nil {
			log.Printf("failed to create machinery client for %s: %v", node.IPAddress, err)
			return
		}

		// --- Start cluster health check ---
		healthCheckClient, err := cli.ClusterHealthCheck(ctx, 20*time.Minute, &clusterapi.ClusterInfo{})
		if err != nil {
			log.Printf("failed to start health check: %v", err)
			return
		}

		// --- Ensure CloseSend won't panic ---
		defer func() {
			if r := recover(); r != nil {
				log.Printf("recovered from panic during CloseSend: %v", r)
			}
			if err := healthCheckClient.CloseSend(); err != nil {
				log.Printf("error closing stream: %v", err)
			}
		}()

		// --- Receive messages ---
		for {
			msg, err := healthCheckClient.Recv()
			if err != nil {
				// graceful exit cases
				if err == io.EOF || machineryClient.StatusCode(err) == codes.Canceled {
					log.Println("health check stream closed gracefully")
					break
				}

				// network / transport errors
				log.Printf("recv error: %v", err)
				break
			}

			// handle message errors
			if metaErr := msg.GetMetadata().GetError(); metaErr != "" {
				log.Printf("cluster health check failed: %s", metaErr)
			}
		}
	},
	JobArgs: []any{
		(*talos.TalosService)(nil), // types to be resolved dynamically
		(*gorm.DB)(nil),
	},
	Options: []gocron.JobOption{
		gocron.WithSingletonMode(gocron.LimitModeWait),
	},
}

var NodeStatusUpdateJob *StolosJob = &StolosJob{
	Name:       "NodeStatusUpdateJob",
	Definition: gocron.DurationJob(30 * time.Second),
	JobFunc: func(ts *talos.TalosService, db *gorm.DB, wsManager *wsservices.Manager) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// --- pick a seed node to contact Talos API ---
		var seed models.Node
		if err := db.Where("status = ?", models.StatusActive).
			Where("ip_address <> ''").
			First(&seed).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("NodeStatusUpdateJob: no ACTIVE node with IP found")
				return
			}
			log.Printf("NodeStatusUpdateJob: db error selecting seed: %v", err)
			return
		}

		cli, err := ts.GetMachineryClient(seed.IPAddress)
		if err != nil {
			log.Printf("NodeStatusUpdateJob: unable to get Talos client for %s: %v", seed.IPAddress, err)
			return
		}

		// --- get Affiliates list from cluster namespace ---
		affs, err := talos.GetTypedTalosResourceList[*clusterres.Affiliate](
			ctx, cli, clusterres.NamespaceName, clusterres.AffiliateType,
		)
		if err != nil {
			log.Printf("NodeStatusUpdateJob: failed to get Affiliates: %v", err)
			return
		}
		if affs.Len() == 0 {
			log.Printf("NodeStatusUpdateJob: no Affiliates returned")
		}

		// Collect affiliate hostnames
		var hostnames []string
		affs.ForEach(func(a *clusterres.Affiliate) {
			spec := a.TypedSpec()
			name := spec.Hostname
			if name == "" {
				name = spec.Nodename
			}
			if name != "" {
				hostnames = append(hostnames, name)
			}
		})

		if len(hostnames) == 0 {
			log.Printf("NodeStatusUpdateJob: no valid hostnames found in Affiliates")
			if err := db.Model(&models.Node{}).
				Where("provider = ?", "onprem").
				Update("status", models.StatusFailed).Error; err != nil {
				log.Printf("NodeStatusUpdateJob: failed to mark missing nodes as failed: %v", err)
			}
		} else {
			if err := db.Model(&models.Node{}).
				Where("name IN ?", hostnames).
				Update("status", models.StatusActive).Error; err != nil {
				log.Printf("NodeStatusUpdateJob: DB update failed: %v", err)
				return
			}

			if err := db.Model(&models.Node{}).
				Where("provider = ?", "onprem").
				Where("name NOT IN ?", hostnames).
				Update("status", models.StatusFailed).Error; err != nil {
				log.Printf("NodeStatusUpdateJob: failed to mark missing nodes as failed: %v", err)
			}
		}

		var nodes []models.Node
		if err := db.Find(&nodes).Error; err != nil {
			log.Printf("NodeStatusUpdateJob: failed to load nodes for broadcast: %v", err)
		}

		if wsManager != nil {
			wsManager.BroadcastToSessionType(wsservices.SessionTypeEvent, wsservices.Message{
				Type: "NodeStatusUpdated",
				Payload: map[string]any{
					"nodes":     nodes,
					"updatedAt": time.Now().UTC(),
				},
			})
		}
	},
	JobArgs: []any{
		(*talos.TalosService)(nil),
		(*gorm.DB)(nil),
		(*wsservices.Manager)(nil),
	},
	Options: nil,
}

// NodeInfoReconciler
//  1. list Affiliates from any reachable ACTIVE node
//  2. if node's IP unchanged -> skip
//     else -> GetMachineBestExternalNetworkInterface + update DB (IP + MAC)
//  3. upsert row by hostname (create if missing)
var NodeInfoReconciler = &StolosJob{
	Name:       "NodeInfoReconciler",
	Definition: gocron.DurationJob(30 * time.Second),
	JobArgs: []any{
		(*talos.TalosService)(nil),
		(*node.NodeService)(nil),
		(*gorm.DB)(nil),
		(*wsservices.Manager)(nil),
	},
	Options: []gocron.JobOption{
		gocron.WithSingletonMode(gocron.LimitModeWait),
	},
	JobFunc: func(ts *talos.TalosService, ns *node.NodeService, db *gorm.DB, wsManager *wsservices.Manager) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		// --- seed: any ACTIVE node with an IP ---
		var seed models.Node
		if err := db.Where("status = ?", models.StatusActive).
			Where("ip_address <> ''").
			First(&seed).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("NodeInfoReconciler: no ACTIVE node with IP found")
				return
			}
			log.Printf("NodeInfoReconciler: db error selecting seed: %v", err)
			return
		}

		seedCli, err := ts.GetMachineryClient(seed.IPAddress)
		if err != nil {
			log.Printf("NodeInfoReconciler: seed client err for %s: %v", seed.IPAddress, err)
			return
		}

		// list Affiliates from the seed (cluster namespace)
		affs, err := talos.GetTypedTalosResourceList[*clusterres.Affiliate](
			ctx, seedCli, clusterres.NamespaceName, clusterres.AffiliateType,
		)
		if err != nil {
			log.Printf("NodeInfoReconciler: get Affiliates failed: %v", err)
			return
		}
		if affs.Len() == 0 {
			log.Printf("NodeInfoReconciler: no Affiliates returned")
			return
		}

		// iterate affiliates
		affs.ForEach(func(a *clusterres.Affiliate) {
			spec := a.TypedSpec()
			hostname := spec.Hostname
			if hostname == "" {
				hostname = spec.Nodename
			}
			if hostname == "" {
				return
			}

			// select an IPv4 from Affiliate addresses (first global-unicast v4, else any v4)
			var newIP string
			for _, ip := range spec.Addresses {
				if ip.Is4() && ip.IsGlobalUnicast() {
					newIP = ip.String()
					break
				}
			}
			if newIP == "" {
				for _, ip := range spec.Addresses {
					if ip.Is4() {
						newIP = ip.String()
						break
					}
				}
			}

			// fetch existing node (by hostname-as-key)
			var existing models.Node
			err := db.Where("name = ?", hostname).First(&existing).Error
			notFound := errors.Is(err, gorm.ErrRecordNotFound)
			if err != nil && !notFound {
				log.Printf("NodeInfoReconciler: db read %s failed: %v", hostname, err)
				return
			}

			// If the node exists and its current IP is still present in the affiliate list -> skip
			if !notFound && existing.IPAddress != "" {
				same := false
				for _, ip := range spec.Addresses {
					if ip.Is4() && ip.String() == existing.IPAddress {
						same = true
						break
					}
				}
				if same {
					// nothing to do
					return
				}
			}

			// IP changed or node not in DB -> resolve best external NIC to get MAC
			var mac string
			if newIP != "" {
				if cli, err := ts.GetMachineryClient(newIP); err == nil {
					iface := talos.GetMachineBestExternalNetworkInterface(ctx, cli)
					mac = iface.Mac
				} else {
					log.Printf("NodeInfoReconciler: client err %s (%s): %v", hostname, newIP, err)
				}
			}

			// prepare upsert payload
			upd := map[string]any{
				"name": hostname,
			}
			if newIP != "" {
				upd["ip_address"] = newIP
			}
			if mac != "" {
				upd["mac_address"] = mac
			}

			// Upsert by name (unique index recommended on "name")
			if err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}}, // conflict target
				DoUpdates: clause.Assignments(upd),
			}).Create(&models.Node{
				Name:       hostname,
				IPAddress:  newIP,
				MACAddress: mac,
			}).Error; err != nil {
				log.Printf("NodeInfoReconciler: upsert %s failed: %v", hostname, err)
				return
			}
		})

		if wsManager != nil {
			var nodes []models.Node
			if err := db.Find(&nodes).Error; err != nil {
				log.Printf("NodeInfoReconciler: failed to load nodes for broadcast: %v", err)
			} else {
				wsManager.BroadcastToSessionType(wsservices.SessionTypeEvent, wsservices.Message{
					Type: "NodeStatusUpdated",
					Payload: map[string]any{
						"nodes":     nodes,
						"updatedAt": time.Now().UTC(),
					},
				})
			}
		}
	},
}
