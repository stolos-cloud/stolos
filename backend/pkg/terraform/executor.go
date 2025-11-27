package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// CheckTerraformInstalled verifies that Terraform is installed and available
func CheckTerraformInstalled() error {
	_, err := exec.LookPath("terraform")
	if err != nil {
		return fmt.Errorf("terraform binary not found in PATH - please install Terraform to use cloud provider features: %w", err)
	}
	return nil
}

type Executor struct {
	workDir string
	tf      *tfexec.Terraform
	envVars map[string]string
}

// creates a new terraform executor for the given working directory
func NewExecutor(workDir string, envVars map[string]string) (*Executor, error) {
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		return nil, fmt.Errorf("terraform binary not found: %w", err)
	}

	tf, err := tfexec.NewTerraform(workDir, terraformPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform instance: %w", err)
	}

	if envVars != nil {
		// Preserve PATH
		if _, ok := envVars["PATH"]; !ok {
			if path := os.Getenv("PATH"); path != "" {
				envVars["PATH"] = path
			}
		}
		tf.SetEnv(envVars)
	}

	return &Executor{
		workDir: workDir,
		tf:      tf,
		envVars: envVars,
	}, nil
}

func (e *Executor) Init(ctx context.Context) error {
	if err := e.tf.Init(ctx); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}
	return nil
}

func (e *Executor) Plan(ctx context.Context) (bool, error) {
	hasChanges, err := e.tf.Plan(ctx)
	if err != nil {
		return false, fmt.Errorf("terraform plan failed: %w", err)
	}
	return hasChanges, nil
}

// PlanWithOutput runs terraform plan and returns the plan output as a string
func (e *Executor) PlanWithOutput(ctx context.Context) (bool, string, error) {
	// Use Plan with output redirect
	planFile := "tfplan.out"
	hasChanges, err := e.tf.Plan(ctx, tfexec.Out(planFile))
	if err != nil {
		return false, "", fmt.Errorf("terraform plan failed: %w", err)
	}

	// Show the plan in human-readable format
	planOutput, err := e.tf.ShowPlanFileRaw(ctx, planFile)
	if err != nil {
		return hasChanges, "", fmt.Errorf("terraform show failed: %w", err)
	}

	return hasChanges, planOutput, nil
}

// GetPlanJSON returns the JSON representation of the last plan
func (e *Executor) GetPlanJSON(ctx context.Context) ([]byte, error) {
	planFile := "tfplan.out"
	plan, err := e.tf.ShowPlanFile(ctx, planFile)
	if err != nil {
		return nil, fmt.Errorf("terraform show -json failed: %w", err)
	}

	// Marshal the plan to JSON
	jsonBytes, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plan: %w", err)
	}

	return jsonBytes, nil
}

// PlanJSON runs terraform plan with JSON output for machine-readable resource tracking
func (e *Executor) PlanJSON(ctx context.Context, w io.Writer) (bool, error) {
	hasChanges, err := e.tf.PlanJSON(ctx, w)
	if err != nil {
		return false, fmt.Errorf("terraform plan failed: %w", err)
	}
	return hasChanges, nil
}

func (e *Executor) Apply(ctx context.Context) error {
	if err := e.tf.Apply(ctx); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}
	return nil
}

// ApplyJSON runs terraform apply with JSON output for machine-readable resource tracking
func (e *Executor) ApplyJSON(ctx context.Context, w io.Writer) error {
	if err := e.tf.ApplyJSON(ctx, w); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}
	return nil
}

func (e *Executor) Destroy(ctx context.Context) error {
	if err := e.tf.Destroy(ctx); err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}
	return nil
}

func (e *Executor) ForceUnlock(ctx context.Context, lockID string) error {
	if err := e.tf.ForceUnlock(ctx, lockID); err != nil {
		return fmt.Errorf("terraform force-unlock failed: %w", err)
	}
	return nil
}

// Output retrieves the outputs of the Terraform state in JSON format
// Had to do a workaround since tfexec.Outputs() does not return outputs
// properly. Would always run into an error like: Unexpected EOF.
func (e *Executor) Output(ctx context.Context) (map[string]tfexec.OutputMeta, error) {
	env := os.Environ()
	if e.envVars != nil {
		for k, v := range e.envVars {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	cmd := exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = e.workDir
	cmd.Env = env

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("terraform output failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse the JSON output ourselves
	var outputs map[string]tfexec.OutputMeta
	if err := json.Unmarshal([]byte(stdout.String()), &outputs); err != nil {
		return nil, fmt.Errorf("failed to parse terraform output: %w", err)
	}

	return outputs, nil
}

func (e *Executor) WorkDir() string {
	return e.workDir
}
