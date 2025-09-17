package main

import (
	"context"
	"fmt"
	"os"
	"time"

	factoryClient "github.com/siderolabs/image-factory/pkg/client"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	talosgen "github.com/siderolabs/talos/cmd/talosctl/cmd/mgmt/gen"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
)

func CreateMachineryClientFromTalosconfig(talosConfig *config.Config) machineryClient.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO : Verify MachineryClient configuration
	machinery, _ := machineryClient.New(
		ctx,
		machineryClient.WithConfig(talosConfig),
		machineryClient.WithGRPCDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	_, err := machinery.ServiceInfo(ctx, "api")
	if err != nil {
		fmt.Println("Error validating machinery api connection:", err)
		return machineryClient.Client{}
	}

	return *machinery
}

func CreateMachineConfigBundle(controlPlaneIp string) (*bundle.Bundle, error) {

	var secretsBundle *secrets.Bundle

	genOptions := []generate.Option{
		generate.WithSecretsBundle(secretsBundle),
		generate.WithNetworkOptions(
			v1alpha1.WithKubeSpan(),
		),
		generate.WithInstallDisk(bootstrapInfos.TalosInstallDisk),
		generate.WithInstallImage("ghcr.io/siderolabs/installer:latest"),
		// generate.WithAdditionalSubjectAltNames([]string{_____}),// TODO : Add the right SAN for external IP / DNS
		generate.WithPersist(true),
		generate.WithClusterDiscovery(true),
	}

	configBundle, err := talosgen.GenerateConfigBundle(
		genOptions,
		bootstrapInfos.ClusterName,
		fmt.Sprintf("https://%s:6443", controlPlaneIp),
		bootstrapInfos.KubernetesVersion,
		[]string{},
		[]string{},
		[]string{})

	if err != nil {
		return nil, err
	}

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
func SaveSplitConfigBundleFiles(logger *UILogger, configBundle bundle.Bundle) error {
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

func ExecuteBootstrap(talosApiClient machineryClient.Client) {

	bootrapRequest := machine.BootstrapRequest{
		RecoverEtcd:          false,
		RecoverSkipHashCheck: false,
	}

	err := talosApiClient.Bootstrap(context.Background(), &bootrapRequest)
	if err != nil {
		panic(fmt.Sprintf("Failed to bootstrap talos: %s", err))
	}
}
