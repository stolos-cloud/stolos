package talos

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	factoryClient "github.com/siderolabs/image-factory/pkg/client"
	eventsapi "github.com/siderolabs/siderolink/api/events"
	"github.com/siderolabs/siderolink/pkg/events"
	clusterapi "github.com/siderolabs/talos/pkg/machinery/api/cluster"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/api/storage"
	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"github.com/siderolabs/talos/pkg/machinery/config/container"
	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
	"github.com/siderolabs/talos/pkg/machinery/proto"
	"github.com/stolos-cloud/stolos-bootstrap/internal/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"

	talosgen "github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/gen"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	machineconf "github.com/siderolabs/talos/pkg/machinery/config/machine"
)

type LogEvent struct {
	Node      string
	Payload   any
	EventType string
}

type EventHandler struct {
	HandleEventFunc func(ctx context.Context, event events.Event) error
}

func (h *EventHandler) HandleEvent(ctx context.Context, event events.Event) error {
	return h.HandleEventFunc(ctx, event)
}

func ApplyConfigsToNodes(machineCache *Machines, machinesDisks MachinesDisks, talosInfos *TalosInfo, configBundle *bundle.Bundle) (*bundle.Bundle, error) {
	var err error

	// CONTROLPLANES
	i := 1
	for ip, conf := range machineCache.ControlPlanes {

		if configBundle == nil {
			configBundle, err = CreateMachineConfigBundle(ip, talosInfos)
			if err != nil {
				return configBundle, err
			}
		}

		if len(conf) > 0 {
			continue
		}

		cfg := &v1alpha1.Config{
			ConfigVersion: "v1alpha1",
			MachineConfig: &v1alpha1.MachineConfig{
				MachineNetwork: &v1alpha1.NetworkConfig{
					NetworkHostname: fmt.Sprintf("controlplane-%d", i),
				},
				MachineInstall: &v1alpha1.InstallConfig{
					InstallDiskSelector: &v1alpha1.InstallDiskSelector{
						BusPath: machinesDisks[ip],
					},
				},
			},
		}

		ctr := container.NewV1Alpha1(cfg)
		patch := configpatcher.NewStrategicMergePatch(ctr)
		err = configBundle.ApplyPatches([]configpatcher.Patch{patch}, true, false)

		machineConfigRendered, err := configBundle.Serialize(encoder.CommentsDocs, machineconf.TypeControlPlane)
		if err != nil {
			return configBundle, err
		}

		c, err := machineryClient.New(context.Background(), machineryClient.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		}), machineryClient.WithEndpoints(ip))

		if err != nil {
			return configBundle, err
		}

		_, err = c.ApplyConfiguration(context.Background(), &machineapi.ApplyConfigurationRequest{
			Data:           machineConfigRendered,
			Mode:           1,
			DryRun:         false,
			TryModeTimeout: nil,
		})

		machineCache.ControlPlanes[ip] = machineConfigRendered
		i++
	}

	//WORKERS
	i = 0
	for ip, conf := range machineCache.Workers {
		if len(conf) > 0 {
			continue
		}

		cfg := &v1alpha1.Config{
			ConfigVersion: "v1alpha1",
			MachineConfig: &v1alpha1.MachineConfig{
				MachineNetwork: &v1alpha1.NetworkConfig{
					NetworkHostname: fmt.Sprintf("worker-%d", i),
				},
				MachineInstall: &v1alpha1.InstallConfig{
					InstallDiskSelector: &v1alpha1.InstallDiskSelector{
						BusPath: machinesDisks[ip],
					},
				},
			},
		}

		ctr := container.NewV1Alpha1(cfg)
		patch := configpatcher.NewStrategicMergePatch(ctr)
		err = configBundle.ApplyPatches([]configpatcher.Patch{patch}, false, true)

		machineConfigRendered, err := configBundle.Serialize(encoder.CommentsDocs, machineconf.TypeWorker)
		if err != nil {
			return configBundle, err
		}

		c, err := machineryClient.New(context.Background(), machineryClient.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		}), machineryClient.WithEndpoints(ip))

		if err != nil {
			return configBundle, err
		}

		_, err = c.ApplyConfiguration(context.Background(), &machineapi.ApplyConfigurationRequest{
			Data:           machineConfigRendered,
			Mode:           1,
			DryRun:         false,
			TryModeTimeout: nil,
		})

		machineCache.Workers[ip] = machineConfigRendered
		i++
	}

	return configBundle, nil
}

func EventSink(talosInfos *TalosInfo, eventHandler func(ctx context.Context, event events.Event) error) error {
	server := grpc.NewServer(
		grpc.SharedWriteBuffer(true),
	)
	var handler events.Adapter = &EventHandler{
		HandleEventFunc: eventHandler,
	}

	sink := events.NewSink(handler, []proto.Message{
		&machineapi.MachineStatusEvent{},
		&machineapi.SequenceEvent{},
		&machineapi.RestartEvent{},
		&machineapi.ConfigLoadErrorEvent{},
		&machineapi.ConfigValidationErrorEvent{},
		&machineapi.AddressEvent{},
		&machineapi.PhaseEvent{},
	})
	eventsapi.RegisterEventSinkServiceServer(server, sink)
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", fmt.Sprintf("%s:%s", talosInfos.HTTPHostname, talosInfos.HTTPPort))
	if err != nil {
		return err
	}
	err = server.Serve(listener)
	if err != nil {
		return err
	}
	return nil
}

func CreateMachineryClientFromTalosconfig(talosConfig *config.Config) machineryClient.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO : Verify MachineryClient configuration
	machinery, err := machineryClient.New(
		ctx,
		machineryClient.WithConfig(talosConfig),
		machineryClient.WithGRPCDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	if err != nil {
		panic(err)
	}

	return *machinery
}

func CreateMachineConfigBundle(controlPlaneIp string, talosInfos *TalosInfo) (*bundle.Bundle, error) {

	var secretsBundle *secrets.Bundle

	genOptions := []generate.Option{
		generate.WithSecretsBundle(secretsBundle),
		generate.WithNetworkOptions(
			v1alpha1.WithKubeSpan(),
		),
		generate.WithInstallImage(fmt.Sprintf("ghcr.io/siderolabs/installer:%s", talosInfos.TalosVersion)),
		// generate.WithAdditionalSubjectAltNames([]string{_____}),// TODO : Add the right SAN for external IP / DNS
		generate.WithPersist(true),
		generate.WithClusterDiscovery(true),
	}

	configBundle, err := talosgen.GenerateConfigBundle(
		genOptions,
		talosInfos.ClusterName,
		fmt.Sprintf("https://%s:6443", controlPlaneIp),
		talosInfos.KubernetesVersion,
		[]string{},
		[]string{},
		[]string{})

	if err != nil {
		return nil, err
	}

	talosConfig := configBundle.TalosConfig().Contexts[talosInfos.ClusterName]
	talosConfig.Endpoints = append(talosConfig.Endpoints, fmt.Sprintf("https://%s:50000", controlPlaneIp))

	return configBundle, nil
}

func CreateFactoryClient() *factoryClient.Client {
	factory, _ := factoryClient.New("https://factory.talos.dev/")
	return factory
}

func ExecuteBootstrap(talosApiClient machineryClient.Client) error {

	bootrapRequest := machine.BootstrapRequest{
		RecoverEtcd:          false,
		RecoverSkipHashCheck: false,
	}

	for {
		err := talosApiClient.Bootstrap(context.Background(), &bootrapRequest)
		if err != nil {
			if !strings.Contains(err.Error(), "connection refused") {
				return err
			}
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	return nil
}

func RunBasicClusterHealthCheck(talosApiClient machineryClient.Client, loggerRef *logging.Logger) {
	healthCheckClient, err := talosApiClient.ClusterHealthCheck(context.Background(), 20*time.Minute, &clusterapi.ClusterInfo{})
	if err != nil {
		panic(err)
	}

	if err := healthCheckClient.CloseSend(); err != nil {
		panic(err)
	}

	for {
		msg, err := healthCheckClient.Recv()
		if err != nil {
			if err == io.EOF || machineryClient.StatusCode(err) == codes.Canceled {
				break
			}
			panic(err)
		}

		if msg.GetMetadata().GetError() != "" {
			(*loggerRef).Errorf("Cluster health check failed: %s", msg.GetMetadata().GetError())
			panic(msg.GetMetadata().GetError())
		}
	}
}

func GetDisks(ctx context.Context, ip string) ([]*storage.Disk, error) {
	c, err := machineryClient.New(ctx, machineryClient.WithTLSConfig(&tls.Config{
		InsecureSkipVerify: true,
	}), machineryClient.WithEndpoints(ip))
	if err != nil {
		return nil, err
	}
	disksRes, err := c.Disks(ctx)
	if err != nil {
		return nil, err
	}
	disks := disksRes.GetMessages()[0].Disks
	return disks, nil
}
