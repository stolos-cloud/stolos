package main

import (
	"context"
	"fmt"
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

/*func GenerateTalosconfig(clusterName, endpoint string) (*config.Config, error) {
	// Pick a version contract (usually current)
	contract := coreconfig.TalosVersionCurrent

	// Create (or load) a secrets bundle — this is what signs client certs
	sec, err := secrets.NewBundle(secrets.NewFixedClock(time.Now()), contract)
	if err != nil {
		return nil, err
	}

	in, err := generate.NewInput(
		clusterName,
		endpoint,
		constants.DefaultKubernetesVersion,
		generate.WithVersionContract(contract),
		generate.WithSecretsBundle(sec),
		// TODO: See if we need to add SANs
		// Optional: add SANs for CP IPs/DNS used by the API server:
		// generate.WithAdditionalSubjectAltNames([]string{"10.0.0.5", "cp.local"}),
	)
	if err != nil {
		return nil, err
	}

	clientCfg, err := in.Talosconfig()
	if err != nil {
		return nil, err
	}

	err = clientCfg.Save(clusterName + "-talosconfig")
	if err != nil {
		fmt.Println("Error saving talosconfig!\n", err)
	}

	return clientCfg, nil
}*/

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
		generate.WithInstallDisk("/dev/sda"),
		generate.WithInstallImage("ghcr.io/siderolabs/installer:latest"),
		// generate.WithAdditionalSubjectAltNames([]string{_____}),// TODO : Add the right SAN for external IP / DNS
		generate.WithPersist(true),
		generate.WithClusterDiscovery(true),
	}

	configBundle, err := talosgen.GenerateConfigBundle(
		genOptions,
		bootstrapInfos.ClusterName,
		fmt.Sprintf("https://%s:443", controlPlaneIp),
		bootstrapInfos.KubernetesVersion,
		[]string{},
		[]string{},
		[]string{})

	if err != nil {
		return nil, err
	}

	return configBundle, nil
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
