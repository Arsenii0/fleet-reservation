package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/arsen/fleet-reservation/deployer/internal/core/ports"
	"github.com/google/uuid"
)

// initMu serialises terraform init calls across all concurrent goroutines.
// The shared plugin cache (TF_PLUGIN_CACHE_DIR) cannot be written by two
// terraform processes simultaneously.

// TODO ArsenP : figure out better approach
var initMu sync.Mutex

const (
	terraformBaseDir = "/reservation/deployer/terraform"
	terraformWorkDir = "/var/lib/fleet-terraform"
)

// TerraformDeployer implements ports.DeploymentPlugin using Terraform.
//
// The module name (e.g. "openclaw-guardian") maps directly to a sub-folder under
// terraformBaseDir.  Each instance gets its own directory under terraformWorkDir
// where .tf sources are copied and state is kept.
type TerraformDeployer struct{}

// Deploy copies the module's .tf files into an instance-specific work directory,
// runs terraform init (if needed) and apply, then returns the connection details.
func (d *TerraformDeployer) Deploy(ctx context.Context, instanceID uuid.UUID, module string) (ports.DeployResult, error) {
	if module == "" {
		return ports.DeployResult{}, fmt.Errorf("terraform plugin: module name is empty")
	}

	instanceDir := filepath.Join(terraformWorkDir, instanceID.String())
	if err := os.MkdirAll(instanceDir, 0o755); err != nil {
		return ports.DeployResult{}, fmt.Errorf("create instance workdir: %w", err)
	}

	srcDir := filepath.Join(terraformBaseDir, module)
	if err := copyTerraformFiles(srcDir, instanceDir); err != nil {
		return ports.DeployResult{}, fmt.Errorf("copy terraform files from %s: %w", srcDir, err)
	}

	// Run init only when providers have not been installed yet.
	// Serialised with initMu to prevent concurrent writes to the shared plugin cache.
	if _, err := os.Stat(filepath.Join(instanceDir, ".terraform", "providers")); os.IsNotExist(err) {
		log.Printf("[terraform] Running init for instance=%s module=%s", instanceID, module)
		initMu.Lock()
		initErr := runTerraform(ctx, instanceDir, "init", "-input=false")
		initMu.Unlock()
		if initErr != nil {
			return ports.DeployResult{}, fmt.Errorf("terraform init: %w", initErr)
		}
	}

	applyArgs := append([]string{"apply", "-auto-approve", "-input=false"}, buildVars(instanceID)...)
	log.Printf("[terraform] Running apply for instance=%s module=%s", instanceID, module)
	if err := runTerraform(ctx, instanceDir, applyArgs...); err != nil {
		return ports.DeployResult{}, fmt.Errorf("terraform apply: %w", err)
	}

	return parseOutputs(ctx, instanceDir)
}

// Destroy runs terraform destroy for the given instance, then removes its work directory.
// If the instance directory was left in a partially-initialised state (e.g. deploy failed
// during init), it re-runs init so that destroy can proceed cleanly.
func (d *TerraformDeployer) Destroy(ctx context.Context, instanceID uuid.UUID, module string) error {
	instanceDir := filepath.Join(terraformWorkDir, instanceID.String())

	if _, err := os.Stat(instanceDir); os.IsNotExist(err) {
		log.Printf("[terraform] No state dir for instance=%s - skipping destroy", instanceID)
		return nil
	}

	// Re-init if providers are missing (covers instances that failed during deploy init).
	if _, err := os.Stat(filepath.Join(instanceDir, ".terraform", "providers")); os.IsNotExist(err) {
		log.Printf("[terraform] Re-initializing before destroy for instance=%s", instanceID)
		initMu.Lock()
		initErr := runTerraform(ctx, instanceDir, "init", "-input=false")
		initMu.Unlock()
		if initErr != nil {
			// Init failed — no AWS resources were ever created, just clean up the local dir.
			log.Printf("[terraform] Init failed before destroy for instance=%s, removing local dir: %v", instanceID, initErr)
			return os.RemoveAll(instanceDir)
		}
	}

	destroyArgs := append([]string{"destroy", "-auto-approve", "-input=false"}, buildVars(instanceID)...)
	log.Printf("[terraform] Running destroy for instance=%s module=%s", instanceID, module)
	if err := runTerraform(ctx, instanceDir, destroyArgs...); err != nil {
		return fmt.Errorf("terraform destroy: %w", err)
	}

	return os.RemoveAll(instanceDir)
}

func buildVars(instanceID uuid.UUID) []string {
	namePrefix := "fleet-" + instanceID.String()[:8]
	vars := []string{"-var", "name_prefix=" + namePrefix}
	if region := os.Getenv("AWS_REGION"); region != "" {
		vars = append(vars, "-var", "aws_region="+region)
	}
	return vars
}

// runTerraform executes a terraform subcommand, streaming stdout/stderr directly
// to the process output so logs appear live in `docker compose` output.
// AWS credentials, TF_PLUGIN_CACHE_DIR etc. are inherited from the environment.
func runTerraform(ctx context.Context, workDir string, args ...string) error {
	log.Printf("[terraform] running: %v", args)
	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %w", args, err)
	}
	return nil
}

// parseOutputs runs `terraform output -json` and extracts IP, username, and password.
func parseOutputs(ctx context.Context, workDir string) (ports.DeployResult, error) {
	cmd := exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	data, err := cmd.Output()
	if err != nil {
		return ports.DeployResult{}, fmt.Errorf("terraform output: %w", err)
	}

	var raw map[string]struct {
		Value interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return ports.DeployResult{}, fmt.Errorf("parse terraform output: %w", err)
	}

	result := ports.DeployResult{}
	if v, ok := raw["instance_public_ip"]; ok {
		result.IPAddress = fmt.Sprintf("%v", v.Value)
	}
	if v, ok := raw["vnc_username"]; ok {
		result.Username = fmt.Sprintf("%v", v.Value)
	}
	if v, ok := raw["vnc_password"]; ok {
		result.Password = fmt.Sprintf("%v", v.Value)
	}
	return result, nil
}

// copyTerraformFiles copies .tf, .sh, and .tfvars files from src to dst.
// Directories (like .terraform/) and state files are intentionally skipped
// so that per-instance state is preserved across re-deploys.
func copyTerraformFiles(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read source dir %s: %w", src, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".tf" && ext != ".sh" && ext != ".tfvars" && ext != ".json" {
			continue
		}
		if err := copyFile(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
