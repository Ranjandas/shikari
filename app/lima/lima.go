package lima

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func ListInstances() []LimaVM {
	cmd := exec.Command("limactl", "list", "--json")

	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}

	var vms []LimaVM

	for _, line := range bytes.Split(output, []byte("\n")) {
		var vm LimaVM
		// condition to avoid duplicate entries
		if string(line) != "" {
			json.Unmarshal([]byte(line), &vm)

			vms = append(vms, vm)
		}
	}
	return vms
}

func GetInstance(name string) LimaVM {

	var vm LimaVM

	vms := ListInstances()

	if len(vms) == 0 {
		return vm
	}

	for _, v := range vms {
		if v.Name == name {
			vm = v
			break
		}
	}

	return vm
}

func GetInstancesByPrefix(name string) []LimaVM {
	var filteredInstances []LimaVM

	instances := ListInstances()

	for _, instance := range instances {
		if strings.HasPrefix(instance.Name, fmt.Sprintf("%s-", name)) {
			filteredInstances = append(filteredInstances, instance)
		}
	}

	return filteredInstances
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

func ExecLimaVM(vmName string, command string, quiet bool) {

	if !quiet {
		fmt.Printf("\nRunning command against: %s\n\n", vmName)
	}

	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("limactl shell %s %s", vmName, command))

	// Set the output to os.Stdout and os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		fmt.Printf("error executing command against VM %s: %v\n", vmName, err)
		return
	}
}

func SpawnLimaVM(vmName string, tmpl string, yqExpression string, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()

	// Define the command to spawn a Lima VM
	limaCmd := fmt.Sprintf("limactl start --name %s %s --tty=false --set '%s'", vmName, tmpl, yqExpression)
	cmd := exec.Command("/bin/sh", "-c", limaCmd)

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

func ShellLimaVM(vmName string) {

	limaCmd := fmt.Sprintf("limactl shell '%s'", vmName)
	cmd := exec.Command("/bin/sh", "-c", limaCmd)

	// Set the input to os.Stdin, output to os.Stdout and os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to get a shell inside %s: %v", vmName, err)
	}

	// Wait for the command to finish and check for errors
	if err := cmd.Wait(); err != nil {
		log.Fatalf("Command finished with error: %v", err)
	}
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

	command := "ip -j addr show dev lima0"
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("limactl shell %s %s", vm.Name, command))

	output, err := cmd.Output()

	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}

	var interfaces []Interface
	err = json.Unmarshal([]byte(output), &interfaces)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
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
