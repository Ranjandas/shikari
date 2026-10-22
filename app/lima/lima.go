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

func StopLimaVM(vmName string) error {
	// Define the command to spawn a Lima VM
	cmd := exec.Command("limactl", "stop", vmName)

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error stopping Lima VM %s: %w", vmName, err)
	}

	fmt.Printf("Lima VM %s stopped successfully.\n", vmName)

	return nil
}

func StartLimaVM(vmName string) error {
	// Define the command to spawn a Lima VM
	cmd := exec.Command("limactl", "start", vmName)

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error starting Lima VM %s: %w", vmName, err)
	}

	fmt.Printf("Lima VM %s started successfully.\n", vmName)

	return nil
}

func DeleteLimaVM(vmName string, force bool) error {
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
		return fmt.Errorf("error deleting Lima VM %s: %w", vmName, err)
	}

	fmt.Printf("Lima VM %s deleted successfully.\n", vmName)

	return nil
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

// SpawnLimaVM creates and starts a Lima VM.
func SpawnLimaVM(vmName string, arch string, tmpl string, yqExpression string) error {
	// Define the command to spawn a Lima VM
	cmd := exec.Command("limactl", "start", "--name", vmName, tmpl, "--arch", arch, "--tty=false", "--set", yqExpression)

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error spawning Lima VM %s: %w", vmName, err)
	}

	fmt.Printf("Lima VM %s spawned successfully.\n", vmName)

	return nil
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

// GetIPAddress returns the address the host can reach the VM on, or "" if
// the VM is not running or has no such address.
func (vm LimaVM) GetIPAddress() string {

	if vm.Status != "Running" {
		return ""
	}

	cmd := exec.Command("limactl", "shell", vm.Name, "ip", "-j", "-4", "addr", "show")

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

	return pickIPAddress(interfaces)
}

// limaUserModeAddress is the address of the VM on Lima's default user-mode
// network, which the host cannot reach.
const limaUserModeAddress = "192.168.5.15"

// pickIPAddress returns the IPv4 address on lima0, else on another lima*
// interface, else on an eth* or en* interface (eg: with vzNAT). Bridges such as
// docker0 or cni0 are skipped, as their addresses are internal to the VM.
func pickIPAddress(interfaces []Interface) string {
	preferences := []func(name string) bool{
		func(name string) bool { return name == "lima0" },
		func(name string) bool { return strings.HasPrefix(name, "lima") },
		func(name string) bool { return strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "en") },
	}

	for _, matches := range preferences {
		for _, iface := range interfaces {
			if !matches(iface.IfName) {
				continue
			}

			for _, addrInfo := range iface.AddrInfo {
				if addrInfo.Family == "inet" && addrInfo.Local != limaUserModeAddress {
					return addrInfo.Local
				}
			}
		}
	}

	return ""
}

// GetIPAddresses returns the address of every VM by name, looking them up in
// parallel.
func GetIPAddresses(vms []LimaVM) map[string]string {
	var mu sync.Mutex
	addresses := make(map[string]string, len(vms))
	byName := make(map[string]LimaVM, len(vms))
	for _, vm := range vms {
		byName[vm.Name] = vm
	}

	// GetIPAddress reports its own errors, and a VM without an address is
	// shown with an empty IP
	_ = RunAll(Names(vms), ipLookupParallelism, 0, func(vmName string) error {
		addr := byName[vmName].GetIPAddress()

		mu.Lock()
		addresses[vmName] = addr
		mu.Unlock()

		return nil
	})

	return addresses
}

// ipLookupParallelism is how many VMs GetIPAddresses asks at once. A lookup is
// a short command, so it does not follow SHIKARI_PARALLELISM.
const ipLookupParallelism = 8
