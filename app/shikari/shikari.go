package shikari

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	lima "github.com/ranjandas/shikari/app/lima"
)

func (c ShikariCluster) GetCurrentVMCount() (uint8, uint8, error) {
	vms, err := lima.GetClusterInstances(c.Name)
	if err != nil {
		return 0, 0, err
	}

	serverCount, clientCount := countVMs(vms)
	return serverCount, clientCount, nil
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

func (c ShikariCluster) CreateCluster(scale bool) error {

	if !c.validateName() {
		return fmt.Errorf("invalid cluster name %q: cluster name can only contain alphanumeric characters", c.Name)
	}

	serverCount, clientCount, err := c.GetCurrentVMCount()
	if err != nil {
		return err
	}

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
			return fmt.Errorf("the following VMs will have to be destroyed: %s. Rerun the command with -f to force the scale down", strings.Join(vmsToDestroy, ","))
		}
	} else {
		// start index from 1 if not a scaling request
		serverVMs = c.generateServerInstanceNames(1, int(c.NumServers))
		clientVMs = c.generateClientInstanceNames(1, int(c.NumClients))
		vmsToCreate = append(vmsToCreate, serverVMs...)
		vmsToCreate = append(vmsToCreate, clientVMs...)
	}

	tmpl, err := templateLocator(c.Template)
	if err != nil {
		return err
	}

	setter := yqSetter{}

	// example: --set '.env.SHIKARI_CLUSTER_NAME = strenv(SHIKARI_YQ_VALUE_0) | ...'
	setter.setEnv("SHIKARI_CLUSTER_NAME", c.Name)

	// record the template, so that scale can create matching VMs
	setter.setEnv("SHIKARI_TEMPLATE", tmpl)

	// append server and client count variables
	targetServers, targetClients := c.targetCounts(scale, serverCount, clientCount)
	setter.setEnv("SHIKARI_SERVER_COUNT", fmt.Sprintf("%d", targetServers))
	setter.setEnv("SHIKARI_CLIENT_COUNT", fmt.Sprintf("%d", targetClients))
	setter.setEnv("SHIKARI_LAUNCH_MODE", launchMode(scale))

	// append user defined environment variable
	if err := c.generateEnvArgs(&setter); err != nil {
		return err
	}

	var errs []error

	if len(vmsToCreate) > 0 {
		if len(c.ImgPath) > 0 {
			absolutePath, err := filepath.Abs(c.ImgPath)
			if err != nil {
				return fmt.Errorf("cannot find the absolute path of the image %s: %w", c.ImgPath, err)
			}

			qcow2, err := isQCOW2(absolutePath)
			if err != nil {
				return fmt.Errorf("cannot read the image %s: %w", absolutePath, err)
			}

			if !qcow2 {
				return fmt.Errorf("image %s is not of type qCOW2", absolutePath)
			}
		}

		var wg sync.WaitGroup
		errCh := make(chan error, len(vmsToCreate))

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

		// Collect any errors reported by the goroutines
		for err := range errCh {
			errs = append(errs, err)
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

		// Collect any errors reported by the goroutines
		for err := range errCh {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
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

func (c ShikariCluster) generateEnvArgs(setter *yqSetter) error {
	envs := c.EnvVars

	for _, e := range envs {
		kv := strings.SplitN(e, "=", 2)

		// never include the value in errors, it can be a license
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return fmt.Errorf("invalid env format for %q, expected key=value", kv[0])
		}

		if err := setter.setEnv(kv[0], kv[1]); err != nil {
			return fmt.Errorf("invalid env format: %w", err)
		}
	}

	return nil
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

// targetCounts returns the number of servers and clients the cluster will
// have once the command completes. When scaling, a count of 0 means the flag
// was not given and that role keeps its current count.
func (c ShikariCluster) targetCounts(scale bool, serverCount, clientCount uint8) (uint8, uint8) {
	servers, clients := c.NumServers, c.NumClients

	if scale {
		if servers == 0 {
			servers = serverCount
		}
		if clients == 0 {
			clients = clientCount
		}
	}

	return servers, clients
}

// templateLocator returns the template argument for `limactl start`. Template
// files are made absolute, so the recorded SHIKARI_TEMPLATE works from any
// directory.
func templateLocator(template string) (string, error) {
	lower := strings.ToLower(template)

	switch {
	case strings.Contains(template, "://"):
		return template, nil
	case strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml"):
		absolutePath, err := filepath.Abs(template)
		if err != nil {
			return "", fmt.Errorf("cannot find the absolute path of the template %s: %w", template, err)
		}
		return absolutePath, nil
	default:
		return fmt.Sprintf("template://%s", template), nil
	}
}

// InheritFrom makes a scale request create VMs like the existing VM of the
// cluster: arch and template are taken from it unless set on the command line.
// It returns a warning when the template could not be inherited.
func (c *ShikariCluster) InheritFrom(vm lima.LimaVM, archSet bool, templateSet bool) string {
	if !archSet && vm.Arch != "" {
		c.Arch = vm.Arch
	}

	if templateSet {
		return ""
	}

	if template := vm.Config.Env["SHIKARI_TEMPLATE"]; template != "" {
		c.Template = template
		return ""
	}

	return fmt.Sprintf("cluster %s does not record the template it was created with, using %s. Pass -t to choose another", c.Name, c.Template)
}

func launchMode(scale bool) string {
	if scale {
		return "scale"
	}

	return "create"
}
