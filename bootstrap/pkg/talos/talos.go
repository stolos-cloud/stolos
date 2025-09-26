package talos

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cosi-project/runtime/pkg/safe"
	factoryClient "github.com/siderolabs/image-factory/pkg/client"
	"github.com/siderolabs/talos/cmd/talosctl/cmd/talos"
	"github.com/siderolabs/talos/pkg/cluster"
	"github.com/siderolabs/talos/pkg/cluster/check"
	clusterapi "github.com/siderolabs/talos/pkg/machinery/api/cluster"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
	clusterres "github.com/siderolabs/talos/pkg/machinery/resources/cluster"
	"github.com/stolos-cloud/stolos-bootstrap/internal/logging"
	"github.com/stolos-cloud/stolos-bootstrap/internal/tui"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/state"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"

	talosgen "github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/gen"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
)

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

func CreateMachineConfigBundle(controlPlaneIp string, bootstrapInfos state.BootstrapInfo) (*bundle.Bundle, error) {

	var secretsBundle *secrets.Bundle

	genOptions := []generate.Option{
		generate.WithSecretsBundle(secretsBundle),
		generate.WithNetworkOptions(
			v1alpha1.WithKubeSpan(),
		),
		generate.WithInstallDisk(bootstrapInfos.TalosInfo.TalosInstallDisk),
		generate.WithInstallImage("ghcr.io/siderolabs/installer:latest"),
		// generate.WithAdditionalSubjectAltNames([]string{_____}),// TODO : Add the right SAN for external IP / DNS
		generate.WithPersist(true),
		generate.WithClusterDiscovery(true),
	}

	configBundle, err := talosgen.GenerateConfigBundle(
		genOptions,
		bootstrapInfos.TalosInfo.ClusterName,
		fmt.Sprintf("https://%s:6443", controlPlaneIp),
		bootstrapInfos.TalosInfo.KubernetesVersion,
		[]string{},
		[]string{},
		[]string{})

	if err != nil {
		return nil, err
	}

	talosConfig := configBundle.TalosConfig().Contexts[bootstrapInfos.TalosInfo.ClusterName]
	talosConfig.Endpoints = append(talosConfig.Endpoints, fmt.Sprintf("https://%s:50000", controlPlaneIp))

	return configBundle, nil
}

// ReadSplitConfigBundleFiles reconstructs multiple yaml configs into a ConfigBundle
func ReadSplitConfigBundleFiles() (*bundle.Bundle, error) {
	//dec := yaml.NewDecoder(bytes.NewReader(bundleBytes))

	configBundleOpts := []bundle.Option{
		//bundle.WithInputOptions(
		//	&bundle.InputOptions{
		//		ClusterName: bootstrapInfos.ClusterName,
		//	},
		//),
		bundle.WithExistingConfigs("./"),
	}

	return bundle.NewBundle(configBundleOpts...)

}

// SaveSplitConfigBundleFiles take a config bundle and saves each composite part to individual files for later loading
func SaveSplitConfigBundleFiles(configBundle bundle.Bundle) error {
	initBytes, err := configBundle.InitCfg.Bytes()
	err = os.WriteFile("init.yaml", initBytes, 0644)
	workerBytes, err := configBundle.WorkerCfg.Bytes()
	err = os.WriteFile("worker.yaml", workerBytes, 0644)
	controlPlaneBytes, err := configBundle.ControlPlaneCfg.Bytes()
	err = os.WriteFile("controlplane.yaml", controlPlaneBytes, 0644)
	talosBytes, err := configBundle.TalosCfg.Bytes()
	err = os.WriteFile("talosconfig", talosBytes, 0644)
	return err
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

	return talosApiClient.Bootstrap(context.Background(), &bootrapRequest)
}

func RunBasicClusterHealthCheck(err error, talosApiClient machineryClient.Client, loggerRef *tui.UILogger) {
	healthCheckClient, err := talosApiClient.ClusterHealthCheck(context.Background(), 20*time.Minute, &clusterapi.ClusterInfo{})
	if err != nil {
		loggerRef.Errorf("Failed to get cluster health: %v", err)
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
			loggerRef.Errorf("Cluster health check failed: %s", msg.GetMetadata().GetError())
			panic(msg.GetMetadata().GetError())
		}
	}
}

// ======================
// Reference: The following code is heavily based on `health.go` part of the talosctl command line utility.
// https://github.com/siderolabs/talos/tree/main/cmd/talosctl/cmd/talos/health.go
func RunDetailedClusterHealthCheck(talosApiClient machineryClient.Client, loggerRef *tui.UILogger) {
	// Create ClientProvider
	clientProvider := &cluster.ConfigClientProvider{
		DefaultClient: &talosApiClient,
	}

	members, err := DiscoverClusterMembers()

	// Create Info
	clusterInfo, err := check.NewDiscoveredClusterInfo(members)

	// Build ClusterInfo for check
	checkClusterInfo := struct {
		cluster.ClientProvider
		cluster.K8sProvider
		cluster.Info
	}{
		ClientProvider: clientProvider,
		K8sProvider: &cluster.KubernetesClient{
			ClientProvider: clientProvider,
		},
		Info: clusterInfo,
	}

	// Run Healthcheck and report to custom logger
	err = check.Wait(context.Background(), &checkClusterInfo, append(check.DefaultClusterChecks(), check.ExtraClusterChecks()...), logging.UILoggerReporter(loggerRef))
	if err != nil {
		loggerRef.Info("Failure running health checks!")
	}
}

func DiscoverClusterMembers() ([]*clusterres.Member, error) {
	// Discover Cluster Members
	var members []*clusterres.Member
	err := talos.WithClientNoNodes(func(ctx context.Context, c *machineryClient.Client) error {
		items, err := safe.StateListAll[*clusterres.Member](ctx, c.COSI)
		if err != nil {
			return err
		}

		items.ForEach(func(item *clusterres.Member) { members = append(members, item) })

		return nil
	})
	return members, err
}

//======================
