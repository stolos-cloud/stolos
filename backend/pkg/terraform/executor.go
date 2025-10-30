package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"

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

func (e *Executor) Apply(ctx context.Context) error {
	if err := e.tf.Apply(ctx); err != nil {
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

func (e *Executor) Output(ctx context.Context) (map[string]tfexec.OutputMeta, error) {
	output, err := e.tf.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform output failed: %w", err)
	}
	return output, nil
}

func (e *Executor) WorkDir() string {
	return e.workDir
}
