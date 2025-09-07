package main

import (
	"context"
	"fmt"
	"time"

	factoryClient "github.com/siderolabs/image-factory/pkg/client"
	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/client/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	coreconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"github.com/siderolabs/talos/pkg/machinery/constants"
)

func GenerateTalosconfig(clusterName, endpoint string) (*config.Config, error) {
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
}

func CreateMachineryClientFromTalosconfig(file string) *machineryClient.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	machinery, _ := machineryClient.New(
		ctx,
		machineryClient.WithConfigFromFile(file),
		machineryClient.WithGRPCDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	_, err := machinery.ServiceInfo(ctx, "api")
	if err != nil {
		fmt.Println("Error validating machinery api connection:", err)
		return nil
	}

	return machinery
}

func CreateFactoryClient() *factoryClient.Client {
	factory, _ := factoryClient.New("https://factory.talos.dev/")
	return factory
}
