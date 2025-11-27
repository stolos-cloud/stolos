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
	"github.com/stolos-cloud/stolos/backend/internal/services/k8s"
	"github.com/stolos-cloud/stolos/backend/internal/services/node"
	"github.com/stolos-cloud/stolos/backend/internal/services/talos"
	wsservices "github.com/stolos-cloud/stolos/backend/internal/services/websocket"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// -- job definitions

var ClusterHealthCheckJob = &StolosJob{
	Name:       "ClusterHealthCheckJob",
	Definition: gocron.DurationJob(1 * time.Minute),
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
	JobFunc: func(ts *talos.TalosService, db *gorm.DB, wsManager *wsservices.Manager, k8sClient *k8s.K8sClient) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var dbNodes []models.Node
		if err := db.Find(&dbNodes).Error; err != nil {
			log.Printf("NodeStatusUpdateJob: failed to load nodes: %v", err)
			return
		}

		if k8sClient == nil || k8sClient.Clientset == nil {
			log.Printf("NodeStatusUpdateJob: Kubernetes client not available")
			return
		}

		k8sNodes, err := k8sClient.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("NodeStatusUpdateJob: failed to list Kubernetes nodes: %v", err)
			return
		}

		statusMap := make(map[string]models.NodeStatus, len(k8sNodes.Items))
		for _, kNode := range k8sNodes.Items {
			statusMap[kNode.Name] = mapK8sNodeStatus(&kNode)
		}

		for i := range dbNodes {
			node := &dbNodes[i]

			if node.Status == models.StatusPending {
				continue
			}

			desiredStatus, exists := statusMap[node.Name]
			if !exists {
				if node.Status == models.StatusProvisioning {
					continue
				}
				desiredStatus = models.StatusFailed
			}

			if node.Status == models.StatusProvisioning && desiredStatus == models.StatusFailed {
				continue
			}

			if node.Status == desiredStatus {
				continue
			}

			if err := db.Model(&models.Node{}).
				Where("id = ?", node.ID).
				Update("status", desiredStatus).Error; err != nil {
				log.Printf("NodeStatusUpdateJob: failed to update status for node %s: %v", node.Name, err)
				continue
			}
			node.Status = desiredStatus
		}

		if wsManager != nil {
			wsManager.BroadcastToSessionType(wsservices.SessionTypeEvent, wsservices.Message{
				Type: "NodeStatusUpdated",
				Payload: map[string]any{
					"nodes":     dbNodes,
					"updatedAt": time.Now().UTC(),
				},
			})
		}
	},
	JobArgs: []any{
		(*talos.TalosService)(nil),
		(*gorm.DB)(nil),
		(*wsservices.Manager)(nil),
		(*k8s.K8sClient)(nil),
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
	Definition: gocron.DurationJob(2 * time.Minute),
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

		seedCli, seedNode, err := ts.GetReachableMachineryClient(ctx)
		if err != nil {
			log.Printf("NodeInfoReconciler: no reachable node found: %v", err)
			return
		}

		if seedNode == nil || seedNode.Provider != "onprem" {
			log.Printf("NodeInfoReconciler: no reachable onprem node available for seed selection")
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
		hostnameList := make([]string, 0)
		affs.ForEach(func(a *clusterres.Affiliate) {
			spec := a.TypedSpec()
			hostname := spec.Hostname
			if hostname == "" {
				hostname = spec.Nodename
			}
			if hostname == "" {
				return
			}

			hostnameList = append(hostnameList, hostname)

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

			if newIP == "" && !notFound {
				newIP = existing.IPAddress
			}

			effectiveIP := newIP
			if effectiveIP == "" && !notFound {
				effectiveIP = existing.IPAddress
			}

			ipChanged := notFound || (newIP != "" && newIP != existing.IPAddress)

			var mac string
			var arch string
			if effectiveIP != "" {
				if cli, err := ts.GetMachineryClient(effectiveIP); err == nil {
					if ipChanged || existing.MACAddress == "" {
						if iface := talos.GetMachineBestExternalNetworkInterface(ctx, cli); iface != nil {
							mac = iface.Mac
						}
					}

					if detectedArch, err := talos.DetectMachineArch(ctx, cli); err == nil && detectedArch != "" {
						if notFound || existing.Architecture == "" || existing.Architecture != detectedArch {
							arch = detectedArch
						}
					}
				} else {
					log.Printf("NodeInfoReconciler: client err %s (%s): %v", hostname, effectiveIP, err)
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
			if arch != "" {
				upd["architecture"] = arch
			}

			// Upsert by name (unique index recommended on "name")
			if err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}}, // conflict target
				DoUpdates: clause.Assignments(upd),
			}).Create(&models.Node{
				Name:         hostname,
				IPAddress:    newIP,
				MACAddress:   mac,
				Architecture: arch,
			}).Error; err != nil {
				log.Printf("NodeInfoReconciler: upsert %s failed: %v", hostname, err)
				return
			}
		})

		// Removed this section as nodes which are shutdown are removed from affiliates. Final logic TBD.
		// Remove nodes no longer present in affiliates
		// if len(hostnameList) == 0 {
		// 	log.Printf("NodeInfoReconciler: warning - no affiliates returned; skipping stale node cleanup")
		// } else {
		// 	if err := db.Where("provider = ?", "onprem").Where("name NOT IN ?", hostnameList).
		// 		Where("status NOT IN ?", []models.NodeStatus{models.StatusPending, models.StatusProvisioning}).
		// 		Delete(&models.Node{}).Error; err != nil {
		// 		log.Printf("NodeInfoReconciler: failed to delete stale nodes: %v", err)
		// 	}
		// }

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

func mapK8sNodeStatus(node *corev1.Node) models.NodeStatus {
	if node == nil {
		return models.StatusFailed
	}

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return models.StatusActive
			}
			return models.StatusFailed
		}
	}

	return models.StatusFailed
}
