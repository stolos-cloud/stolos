package talos

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	machineryClient "github.com/siderolabs/talos/pkg/machinery/client"
	netres "github.com/siderolabs/talos/pkg/machinery/resources/network"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
	"github.com/stolos-cloud/stolos/backend/internal/models"
)

// BuildNodeModelFromResources inspects MachineStatus (stage) & LinkStatus (MAC) and builds a minimal Node.
func BuildNodeModelFromResources(ctx context.Context, c *machineryClient.Client, nodeIP string) (*models.Node, error) {
	node := &models.Node{
		IPAddress: nodeIP,
		Provider:  "onprem",
		Status:    models.StatusPending,
	}

	node.MACAddress = GetMachineBestExternalMacCandidate(ctx, c)

	return node, nil
}

func GetTypedTalosResourceList[T resource.Resource](
	ctx context.Context,
	c *machineryClient.Client,
	namespace string,
	typ string,
	opts ...state.ListOption,
) (safe.List[T], error) {
	rd, err := c.ResolveResourceKind(ctx, &namespace, typ) // Find full resource type if an alias was used.
	if err != nil {
		return safe.List[T]{}, fmt.Errorf("resolve kind %s/%s: %w", namespace, typ, err)
	}

	//metadata for the output list
	md := resource.NewMetadata(namespace, rd.TypedSpec().Type, "", resource.VersionUndefined)

	listCtx, cancel := context.WithTimeout(ctx, 5*time.Second) //TODO : see if timeout for listing needs adjusting
	defer cancel()

	lst, err := safe.StateList[T](listCtx, c.COSI, md, opts...)
	if err != nil {
		return safe.List[T]{}, fmt.Errorf("list %s/%s: %w", namespace, rd.TypedSpec().Type, err)
	}
	return lst, nil
}

// GetTypedTalosResource resolves <namespace,type,id> and returns a single typed resource T.
// T must be a concrete Talos resource type (pointer).
func GetTypedTalosResource[T resource.Resource](
	ctx context.Context,
	c *machineryClient.Client,
	namespace string,
	typ string,
	id string,
	opts ...state.GetOption,
) (T, error) {
	var zero T

	rd, err := c.ResolveResourceKind(ctx, &namespace, typ)
	if err != nil {
		return zero, fmt.Errorf("resolve kind %s/%s: %w", namespace, typ, err)
	}

	md := resource.NewMetadata(namespace, rd.TypedSpec().Type, id, resource.VersionUndefined)

	getCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := safe.StateGet[T](getCtx, c.COSI, md, opts...)
	if err != nil {
		return zero, fmt.Errorf("get %s/%s/%s: %w", namespace, rd.TypedSpec().Type, id, err)
	}
	return res, nil
}

// GetMachineBestExternalMacCandidate tries to find the external mac address of primary net interface
func GetMachineBestExternalMacCandidate(ctx context.Context, c *machineryClient.Client) string {
	if linkList, err := GetTypedTalosResourceList[*netres.LinkStatus](ctx, c, netres.NamespaceName, "link"); err == nil {
		type cand struct {
			mac   string
			score int
		}
		var best cand

		for link := range linkList.All() {
			spec := link.TypedSpec()

			iface := link.Metadata().ID()
			mac := net.HardwareAddr(spec.HardwareAddr).String()
			if iface == "" || mac == "" || mac == "00:00:00:00:00:00" || isVirtualIface(iface) {
				continue
			}

			score := 0
			if spec.LinkState {
				score += 10
			}
			if strings.ToLower(spec.OperationalState.String()) == "up" {
				score += 5
			}
			if strings.HasPrefix(iface, "en") {
				score += 2
			}

			if score > best.score {
				best = cand{mac: strings.ToLower(mac), score: score}
			}
		}
		return best.mac
	}
	return ""
}

// GetMachineStatus gets the COSI runtime machine status and stage
func GetMachineStatus(c *machineryClient.Client) (*runtime.MachineStatusSpec, error) {
	res, err := GetTypedTalosResource[*runtime.MachineStatus](context.Background(), c, runtime.NamespaceName, runtime.MachineStatusType, "machine")
	if err != nil {
		return nil, err
	}
	spec := res.TypedSpec()
	return spec, nil
}

// isVirtualIface tries to see if the interface is virtual based on the prefix
func isVirtualIface(name string) bool {
	for _, p := range []string{"lo", "bond", "br", "veth", "docker", "cni", "flannel", "kube", "wg", "tun", "tap", "teql", "sit", "ip6tnl", "dummy"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// DetectMachineArch tries to detect cpu arch via /proc/cpuinfo , returns goarch formatted string.
func DetectMachineArch(ctx context.Context, cli *machineryClient.Client) (string, error) {
	// set a timeout to avoid hangs
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rc, err := cli.Read(ctx, "/proc/cpuinfo")
	if err != nil {
		return "", fmt.Errorf("read /proc/cpuinfo: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("readAll /proc/cpuinfo: %w", err)
	}
	text := strings.ToLower(string(data))

	// fast checks
	if strings.Contains(text, "aarch64") || strings.Contains(text, "armv8") {
		return "arm64", nil
	}
	if strings.Contains(text, "x86_64") {
		return "amd64", nil
	}
	if strings.Contains(text, "riscv64") || strings.Contains(text, "rv64") {
		return "riscv64", nil
	}
	if strings.Contains(text, "armv7") || strings.Contains(text, "v7l") {
		return "armv7", nil
	}

	return "Unknown", nil
}
