package lima

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func ListInstances() ([]LimaVM, error) {
	cmd := exec.Command("limactl", "list", "--json")
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Lima VMs: %w", err)
	}

	return parseInstances(output)
}

// parseInstances parses the output of `limactl list --json`, one VM per line.
func parseInstances(output []byte) ([]LimaVM, error) {
	var vms []LimaVM

	for _, line := range bytes.Split(output, []byte("\n")) {
		var vm LimaVM
		// condition to avoid duplicate entries
		if len(bytes.TrimSpace(line)) != 0 {
			if err := json.Unmarshal(line, &vm); err != nil {
				return nil, fmt.Errorf("failed to parse limactl list output: %w", err)
			}

			vms = append(vms, vm)
		}
	}
	return vms, nil
}

func GetInstance(name string) (LimaVM, error) {

	var vm LimaVM

	vms, err := ListInstances()
	if err != nil {
		return vm, err
	}

	for _, v := range vms {
		if v.Name == name {
			vm = v
			break
		}
	}

	return vm, nil
}

func GetInstancesByPrefix(name string) ([]LimaVM, error) {
	var filteredInstances []LimaVM

	instances, err := ListInstances()
	if err != nil {
		return nil, err
	}

	for _, instance := range instances {
		if strings.HasPrefix(instance.Name, fmt.Sprintf("%s-", name)) {
			filteredInstances = append(filteredInstances, instance)
		}
	}

	return filteredInstances, nil
}

func GetInstancesByStatus(instances []LimaVM, status string) []LimaVM {
	var filteredInstances []LimaVM

	for _, instance := range instances {
		if strings.ToLower(instance.Status) == status {
			filteredInstances = append(filteredInstances, instance)
		}
	}

	return filteredInstances
}

func StopLimaVM(vmName string, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()

	// Define the command to spawn a Lima VM
	cmd := exec.Command("limactl", "stop", vmName)

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		errCh <- fmt.Errorf("error stopping Lima VM %s: %w", vmName, err)
		return
	}

	fmt.Printf("Lima VM %s stopped successfully.\n", vmName)
}

func StartLimaVM(vmName string, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()

	// Define the command to spawn a Lima VM
	cmd := exec.Command("limactl", "start", vmName)

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		errCh <- fmt.Errorf("error starting Lima VM %s: %w", vmName, err)
		return
	}

	fmt.Printf("Lima VM %s started successfully.\n", vmName)
}

func DeleteLimaVM(vmName string, force bool, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()

	cmd := exec.Command("limactl", "delete", vmName)

	if force {
		// Force destroy the VMs
		cmd = exec.Command("limactl", "delete", "-f", vmName)
	}

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		errCh <- fmt.Errorf("error deleting Lima VM %s: %w", vmName, err)
		return
	}

	fmt.Printf("Lima VM %s deleted successfully.\n", vmName)
}

// ExecLimaVM runs a command inside the VM. A single argument is a command line
// for a shell inside the VM, eg: 'cat /etc/hosts | grep srv'. Multiple
// arguments are run as-is, keeping their quoting, eg: sh -c 'echo $HOSTNAME'.
// Nothing is ever interpreted by a shell on the host.
func ExecLimaVM(vmName string, args []string, quiet bool) error {

	if !quiet {
		fmt.Printf("\nRunning command against: %s\n\n", vmName)
	}

	cmd := exec.Command("limactl", execArgs(vmName, args)...)

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error executing command against VM %s: %w", vmName, err)
	}

	return nil
}

// execArgs returns the limactl arguments to run args inside the VM.
func execArgs(vmName string, args []string) []string {
	if len(args) == 1 {
		return []string{"shell", vmName, "sh", "-c", args[0]}
	}

	// limactl shell stops parsing its own flags at the VM name, and quotes
	// every argument for the guest shell
	return append([]string{"shell", vmName}, args...)
}

// SpawnLimaVM creates and starts a Lima VM. env is added to the environment of
// limactl, so that the yq expression can read values with strenv().
func SpawnLimaVM(vmName string, arch string, tmpl string, yqExpression string, env []string, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()

	// Define the command to spawn a Lima VM
	cmd := exec.Command("limactl", "start", "--name", vmName, tmpl, "--arch", arch, "--tty=false", "--set", yqExpression)
	cmd.Env = append(os.Environ(), env...)

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		errCh <- fmt.Errorf("error spawning Lima VM %s: %w", vmName, err)
		return
	}

	fmt.Printf("Lima VM %s spawned successfully.\n", vmName)
}

func ShellLimaVM(vmName string) error {

	cmd := exec.Command("limactl", "shell", vmName)

	// Set the input to os.Stdin, output to os.Stdout and os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to get a shell inside %s: %w", vmName, err)
	}

	// The exit status of the shell session is the user's business, eg: the
	// last command typed. Only report failures to start the session.
	cmd.Wait()

	return nil
}

func (vm LimaVM) GetScenarioNameFromEnv() string {

	scenario_name := ""

	if len(vm.Config.Env) == 0 {
		scenario_name = ""
	}

	if env, ok := vm.Config.Env["SHIKARI_SCENARIO_NAME"]; ok {
		scenario_name = env
	}

	return scenario_name
}

func (vm LimaVM) GetVMDir() string {
	return vm.Dir
}

func (vm LimaVM) GetIPAddress() string {

	if vm.Status != "Running" {
		return ""
	}

	cmd := exec.Command("limactl", "shell", vm.Name, "ip", "-j", "addr", "show", "dev", "lima0")

	output, err := cmd.Output()

	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting the IP address of %s: %v\n", vm.Name, err)
		return ""
	}

	var interfaces []Interface
	err = json.Unmarshal([]byte(output), &interfaces)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing the IP address of %s: %v\n", vm.Name, err)
		return ""
	}

	// Extract the local address where family is "inet"
	for _, iface := range interfaces {
		for _, addrInfo := range iface.AddrInfo {
			if addrInfo.Family == "inet" {
				return addrInfo.Local
			}
		}
	}
	return ""
}
