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

	var serverCount, clientCount uint8

	vms := lima.GetInstancesByPrefix(c.Name)

	if len(vms) > 0 {
		for _, vm := range vms {
			if strings.HasPrefix(vm.Name, fmt.Sprintf("%s-cli", c.Name)) {
				clientCount++
			} else {
				serverCount++
			}
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

	userDefinedEnvs := c.generateEnvArgs()

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

		// example: --set '. |= .env.SHIKARI_VM_MODE="server", .env.SHIKARI_CLUSTER_NAME="murphy"'
		yqExpression := fmt.Sprintf(`.env.SHIKARI_CLUSTER_NAME="%s"`, c.Name)

		// append server and client count variables
		countEnvVars := fmt.Sprintf(`.env.SHIKARI_SERVER_COUNT="%d" | .env.SHIKARI_CLIENT_COUNT="%d"`, c.NumServers, c.NumClients)
		yqExpression = fmt.Sprintf("%s |  %s", yqExpression, countEnvVars)

		// append user defined environment variable
		if userDefinedEnvs != "" {
			yqExpression = fmt.Sprintf("%s | %s", yqExpression, userDefinedEnvs)
		}

		// // Override the image from the template
		if len(c.ImgPath) > 0 {
			absolutePath, _ := filepath.Abs(c.ImgPath)
			imageArg := fmt.Sprintf(`.images=[{"location": "%s"}]`, absolutePath)
			yqExpression = fmt.Sprintf("%s | %s", yqExpression, imageArg)
		}

		// Spawn Lima VMs concurrently
		for _, vmName := range vmsToCreate {
			wg.Add(1)
			yqExpr := fmt.Sprintf(`%s | .env.SHIKARI_VM_MODE="%s"`, yqExpression, c.getInstanceMode(vmName))

			go lima.SpawnLimaVM(vmName, tmpl, yqExpr, &wg, errCh)
			yqExpr = ""

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

func (c ShikariCluster) generateEnvArgs() string {
	envs := c.EnvVars

	var envCSV []string
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

		envCSV = append(envCSV, fmt.Sprintf(".env.%s=\"%s\"", kv[0], kv[1]))
	}

	return strings.Join(envCSV, "| ")
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
