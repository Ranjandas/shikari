package shikari

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	lima "github.com/ranjandas/shikari/app/lima"
)

func (c ShikariCluster) GetCurrentVMCount() (uint8, uint8) {
	return countVMs(lima.GetClusterInstances(c.Name))
}

func countVMs(vms []lima.LimaVM) (uint8, uint8) {

	var serverCount, clientCount uint8

	for _, vm := range vms {
		if vm.IsClient() {
			clientCount++
		} else if vm.IsServer() {
			serverCount++
		}
	}

	return serverCount, clientCount
}

func (c ShikariCluster) CreateCluster(scale bool) {

	if !c.validateName() {
		fmt.Println("Cluster name can only contain alphanumeric characters!")
		return
	}

	serverCount, clientCount := c.GetCurrentVMCount()

	var serverVMs, clientVMs []string
	var vmsToCreate, vmsToDestroy []string
	var serverScaleDown, clientScaleDown bool

	if scale {
		// If request > existing count, generate server instance name from the existing count
		if c.NumServers > serverCount {
			serverVMs = c.generateServerInstanceNames(int(serverCount)+1, int(c.NumServers))
			vmsToCreate = append(vmsToCreate, serverVMs...)
		}

		if c.NumServers < serverCount && c.NumServers != 0 {
			serverVMs = c.generateServerInstanceNames(int(c.NumServers)+1, int(serverCount))
			serverScaleDown = true
			vmsToDestroy = append(vmsToDestroy, serverVMs...)
		}

		if c.NumClients > clientCount {
			clientVMs = c.generateClientInstanceNames(int(clientCount)+1, int(c.NumClients))
			vmsToCreate = append(vmsToCreate, clientVMs...)
		}

		if c.NumClients < clientCount && c.NumClients != 0 {
			clientVMs = c.generateClientInstanceNames(int(c.NumClients)+1, int(clientCount))
			vmsToDestroy = append(vmsToDestroy, clientVMs...)
			clientScaleDown = true
		}

		if (clientScaleDown || serverScaleDown) && !c.Force {
			fmt.Println("The following VMs", strings.Join(vmsToDestroy, ","), "will have to be destroyed. Rerun the command with -f to force the scale down!")
			return
		}
	} else {
		// start index from 1 if not a scaling request
		serverVMs = c.generateServerInstanceNames(1, int(c.NumServers))
		clientVMs = c.generateClientInstanceNames(1, int(c.NumClients))
		vmsToCreate = append(vmsToCreate, serverVMs...)
		vmsToCreate = append(vmsToCreate, clientVMs...)
	}

	setter := yqSetter{}

	// example: --set '.env.SHIKARI_CLUSTER_NAME = strenv(SHIKARI_YQ_VALUE_0) | ...'
	setter.setEnv("SHIKARI_CLUSTER_NAME", c.Name)

	// append server and client count variables
	setter.setEnv("SHIKARI_SERVER_COUNT", fmt.Sprintf("%d", c.NumServers))
	setter.setEnv("SHIKARI_CLIENT_COUNT", fmt.Sprintf("%d", c.NumClients))
	setter.setEnv("SHIKARI_LAUNCH_MODE", launchMode(scale))

	// append user defined environment variable
	c.generateEnvArgs(&setter)

	if len(vmsToCreate) > 0 {
		if len(c.ImgPath) > 0 {
			absolutePath, err := filepath.Abs(c.ImgPath)
			if err != nil {
				fmt.Printf("EROR: Cannot find the absolute path of the image: %s", c.ImgPath)
				return
			}

			qcow2, _ := isQCOW2(absolutePath)

			if !qcow2 {
				fmt.Printf("Error: Image %s is not of type qCOW2\n", absolutePath)
				return
			}
		}

		var wg sync.WaitGroup
		errCh := make(chan error, len(vmsToCreate))

		var tmpl string

		if strings.HasSuffix(strings.ToLower(c.Template), ".yml") || strings.HasSuffix(strings.ToLower(c.Template), ".yaml") {
			tmpl = c.Template
		} else {
			tmpl = fmt.Sprintf("template://%s", c.Template)
		}

		// // Override the image from the template
		if len(c.ImgPath) > 0 {
			absolutePath, _ := filepath.Abs(c.ImgPath)
			setter.exprs = append(setter.exprs, ".images = [{}]")
			setter.set(".images[0].location", absolutePath)
		}

		// Spawn Lima VMs concurrently
		for _, vmName := range vmsToCreate {
			wg.Add(1)
			vmSetter := setter.clone()
			vmSetter.setEnv("SHIKARI_VM_MODE", c.getInstanceMode(vmName))

			go lima.SpawnLimaVM(vmName, c.Arch, tmpl, vmSetter.expression(), vmSetter.env, &wg, errCh)
		}

		// Wait for all goroutines to finish
		wg.Wait()

		// Close error channel after all goroutines are done
		close(errCh)

		// Check for any errors reported by the goroutines
		for err := range errCh {
			fmt.Println(err)
		}
	}

	if len(vmsToDestroy) > 0 {
		var wg sync.WaitGroup
		errCh := make(chan error, len(vmsToDestroy))

		for _, vmName := range vmsToDestroy {
			wg.Add(1)
			go lima.DeleteLimaVM(vmName, c.Force, &wg, errCh)
			time.Sleep(2 * time.Second)
		}
		// Wait for all goroutines to finish
		wg.Wait()

		// Close error channel after all goroutines are done
		close(errCh)

		// Check for any errors reported by the goroutines
		for err := range errCh {
			fmt.Println(err)
		}
	}

}

func (c ShikariCluster) generateServerInstanceNames(start int, end int) []string {

	s := make([]string, 0)

	for n := start; n <= end; n++ {
		name := fmt.Sprintf("%s-srv-%02d", c.Name, n)

		s = append(s, name)
	}
	return s
}

func (c ShikariCluster) generateClientInstanceNames(start int, end int) []string {

	s := make([]string, 0)

	for n := start; n <= end; n++ {
		name := fmt.Sprintf("%s-cli-%02d", c.Name, n)

		s = append(s, name)
	}
	return s
}

func (c ShikariCluster) generateEnvArgs(setter *yqSetter) {
	envs := c.EnvVars

	for _, e := range envs {
		if !strings.Contains(e, "=") {
			fmt.Println("Invalid env format")
			os.Exit(1)
		}

		kv := strings.SplitN(e, "=", 2)

		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			fmt.Println("Invalid env format")
			os.Exit(1)
		}

		if err := setter.setEnv(kv[0], kv[1]); err != nil {
			fmt.Println("Invalid env format:", err)
			os.Exit(1)
		}
	}
}

func isQCOW2(filePath string) (bool, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read the first 4 bytes
	header := make([]byte, 4)
	_, err = file.Read(header)
	if err != nil {
		return false, err
	}

	// Check if the header matches the QCOW2 magic number
	qcow2Magic := []byte{'Q', 'F', 'I', 0xfb}
	if string(header) == string(qcow2Magic) {
		return true, nil
	}

	return false, nil
}

func (c ShikariCluster) getInstanceMode(instanceName string) string {
	mode := "server"

	if strings.HasPrefix(instanceName, fmt.Sprintf("%s-cli-", c.Name)) {
		mode = "client"
	}

	return mode
}

func (c ShikariCluster) validateName() bool {
	pattern := `^([a-zA-Z0-9]+)$`
	regex, _ := regexp.Compile(pattern)

	return regex.MatchString(c.Name)
}

func launchMode(scale bool) string {
	if scale {
		return "scale"
	}

	return "create"
}
