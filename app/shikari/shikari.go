package shikari

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	lima "github.com/ranjandas/shikari/app/lima"
)

func (c ShikariCluster) CreateCluster(scale bool) error {

	if !c.validateName() {
		return fmt.Errorf("invalid cluster name %q: cluster name can only contain alphanumeric characters", c.Name)
	}

	instances, err := lima.GetClusterInstances(c.Name)
	if err != nil {
		return err
	}

	existingServers := lima.GetServerInstances(instances)
	existingClients := lima.GetClientInstances(instances)
	serverCount, clientCount := uint8(len(existingServers)), uint8(len(existingClients))

	var vmsToCreate, vmsToDestroy []string

	if scale {
		targetServers, targetClients := c.targetCounts(scale, serverCount, clientCount)
		if targetServers == 0 && targetClients == 0 {
			return fmt.Errorf("scaling cluster %s to 0 servers and 0 clients would remove all its VMs, use destroy instead", c.Name)
		}

		serversToCreate, serversToDestroy := c.planRole("srv", existingServers, targetServers)
		clientsToCreate, clientsToDestroy := c.planRole("cli", existingClients, targetClients)

		vmsToCreate = append(serversToCreate, clientsToCreate...)
		vmsToDestroy = append(serversToDestroy, clientsToDestroy...)

		if len(vmsToDestroy) > 0 && !c.Force {
			return fmt.Errorf("the following VMs will have to be destroyed: %s. Rerun the command with -f to force the scale down", strings.Join(vmsToDestroy, ","))
		}
	} else {
		// start index from 1 if not a scaling request
		vmsToCreate = append(vmsToCreate, c.generateServerInstanceNames(1, int(c.NumServers))...)
		vmsToCreate = append(vmsToCreate, c.generateClientInstanceNames(1, int(c.NumClients))...)
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

	// name the scenario after the template, unless the template names it
	setter.setEnvDefault("SHIKARI_SCENARIO_NAME", scenarioName(tmpl))

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

			// The image format (qcow2, raw, ...) is left to Lima to validate
			if err := checkImage(absolutePath); err != nil {
				return err
			}
		}

		// // Override the image from the template
		if len(c.ImgPath) > 0 {
			absolutePath, _ := filepath.Abs(c.ImgPath)
			setter.exprs = append(setter.exprs, ".images = [{}]")
			setter.set(".images[0].location", absolutePath)
		}

		// A spawn that fails is not retried, as the half-created VM would make the
		// retry fail with a different error
		err := lima.RunAll(vmsToCreate, lima.Parallelism(), 0, func(vmName string) error {
			vmSetter := setter.clone()
			vmSetter.setEnv("SHIKARI_VM_MODE", c.getInstanceMode(vmName))

			return lima.SpawnLimaVM(vmName, c.Arch, tmpl, vmSetter.expression(), vmSetter.env)
		})
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(vmsToDestroy) > 0 {
		err := lima.RunAll(vmsToDestroy, lima.Parallelism(), 1, func(vmName string) error {
			return lima.DeleteLimaVM(vmName, c.Force)
		})
		if err != nil {
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

// checkImage returns an error if the image file can't be used, eg: a typo in
// --image.
func checkImage(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot use the image: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("cannot use the image %s: it is a directory", path)
	}

	return nil
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
// have once the command completes. When scaling, a role whose count was not
// given keeps its current count.
func (c ShikariCluster) targetCounts(scale bool, serverCount, clientCount uint8) (uint8, uint8) {
	servers, clients := c.NumServers, c.NumClients

	if scale {
		if !c.ServersSet {
			servers = serverCount
		}
		if !c.ClientsSet {
			clients = clientCount
		}
	}

	return servers, clients
}

// planRole returns the names of the VMs to create and to destroy to bring a
// role ("srv" or "cli") from the existing VMs to target VMs. New VMs take the
// lowest free numbers, so gaps left by deleted VMs are filled instead of
// colliding with existing names. Scaling down removes the highest numbers.
func (c ShikariCluster) planRole(role string, existing []lima.LimaVM, target uint8) ([]string, []string) {
	var toCreate, toDestroy []string

	used := make(map[int]bool)
	for _, vm := range existing {
		used[vm.Index()] = true
	}

	for n := 1; len(existing)+len(toCreate) < int(target); n++ {
		if !used[n] {
			toCreate = append(toCreate, fmt.Sprintf("%s-%s-%02d", c.Name, role, n))
		}
	}

	if len(existing) > int(target) {
		sorted := append([]lima.LimaVM{}, existing...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Index() > sorted[j].Index() })

		for _, vm := range sorted[:len(existing)-int(target)] {
			toDestroy = append(toDestroy, vm.Name)
		}
	}

	return toCreate, toDestroy
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

// scenarioName returns the scenario name for a template: its file name
// without the extension, eg: nomad for template://nomad or ./nomad.yaml.
// Templates of shikari-scenarios are all named hashibox.yaml, so those are
// named after their directory, eg: nomad-consul-quickstart for
// scenarios/nomad-consul-quickstart/hashibox.yaml.
func scenarioName(template string) string {
	if _, rest, found := strings.Cut(template, "://"); found {
		template = rest
	}

	template = strings.TrimRight(template, "/")
	name := path.Base(template)
	name = strings.TrimSuffix(name, path.Ext(name))

	if name == "hashibox" {
		if dir := path.Base(path.Dir(template)); dir != "." && dir != "/" {
			return dir
		}
	}

	return name
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
