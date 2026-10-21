# shikari

Shikari is an opinionated tool that helps you create multi-vm clusters (primarily aimed at Nomad and Consul) using [Lima](https://lima-vm.io) as the underlying VM Manager.

Various scenarios built for Shikari is available here: https://github.com/Ranjandas/shikari-scenarios/

## Prerequisites

The following are the pre-requisites for using Shikari

* Lima: Used for running VMs
* QEMU [Optional]: Required if you are also baking images using packer
* HashiCorp Packer: To build custom images (available in the shikari-scenarios/packer repository)
* [socket_vmnet](https://github.com/lima-vm/socket_vmnet): Installed and configured as per Lima [requirements](https://lima-vm.io/docs/config/network/#socket_vmnet)
* [Shikari Scenarios](https://github.com/Ranjandas/shikari-scenarios/): This is the primary source of various re-usable scenarios to use with Shikari

## Install

The Shikari binaries are available to download from GH releases. Please download the binary from here: https://github.com/Ranjandas/shikari/releases/latest

## Usage

Shikari can be used to create clusters of any size depending on the capacity of the host on which the VMs are provisioned. Shikari under the hood invokes Lima commands to provision VMs.

The following sub-sections (named after various subcommands) shows how to use Shikari to create, use and destroy clusters.

### Create

The `create` command is used to create clusters. Shikari treats VM instances as either a `server` or a `client`. Please note that shikari as such won't install or configure the applications on the VMs. Shikari only creates the VMs, and the configuration of applications inside the VMs are done using the provisioning scripts included in the scenarios directory.

Here is an example of how Shikari is used to provision a 3x3 (ie 3 servers and 3 clients) Consul and Nomad Cluster.

```
$ shikari create \
    --name murphy \
    --servers 3 \
    --clients 3 \
    --image ../../packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2 \
    --template scenarios/nomad-consul-quickstart/hashibox.yaml \
    --env CONSUL_LICENSE=$(cat consul.hclic) \
    --env NOMAD_LICENSE=$(cat nomad.hclic)
```

If available, license environment variables will be automatically configured. See [License Auto-Loading](#license-auto-loading) for more details.

### List

The `list` command is used to list the clusters and their VMs. You can get VMs of a specific cluster by passing the `--name/-n` flag.

```
$ shikari list -n murphy
CLUSTER    VM NAME          ARCH       IP                STATUS     SCENARIO                   DISK(GB)    MEMORY(GB)    CPUS    IMAGE
murphy     murphy-cli-01    aarch64    192.168.105.10    Running    nomad-consul-quickstart    100         4             4       /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy     murphy-cli-02    aarch64    192.168.105.11    Running    nomad-consul-quickstart    100         4             4       /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy     murphy-cli-03    aarch64    192.168.105.12    Running    nomad-consul-quickstart    100         4             4       /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy     murphy-srv-01    aarch64    192.168.105.13    Running    nomad-consul-quickstart    100         4             4       /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy     murphy-srv-02    aarch64    192.168.105.14    Running    nomad-consul-quickstart    100         4             4       /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy     murphy-srv-03    aarch64    192.168.105.15    Running    nomad-consul-quickstart    100         4             4       /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
```

`IP` is the address the host can reach the VM on, and is empty for stopped VMs. `SCENARIO` comes from the `SHIKARI_SCENARIO_NAME` env variable of the template. Pass `--no-header` to leave out the header line, eg: for scripts.

#### Helper Variables

When spinning up the VMs, Shikari injects a few environment variables into each VM, which would give some additional context to the provisioning scripts that would include:

* What kind of node the script is running on (injected as `SHIKARI_VM_MODE` env variable)
* Name of the cluster (injected as `SHIKARI_CLUSTER_NAME` env variable)
* Count of Number of Servers (injected as `SHIKARI_SERVER_COUNT` env variable)
* Count of Number of Clients (injected as `SHIKARI_CLIENT_COUNT` env variable)

> NOTE: `SHIKARI_SERVER_COUNT` and `SHIKARI_CLIENT_COUNT` are the size of the cluster when the VM was created. On `scale`, new VMs get the new size (a role that wasn't passed to `scale` keeps its current count), while existing VMs keep the counts they were created with.
* Launch mode of VMs (`create` or `scale`) (injected as `SHIKARI_LAUNCH_MODE` env variable)
* Template the cluster was created with (injected as `SHIKARI_TEMPLATE` env variable, and reused by `scale` unless `-t` is passed)

> NOTE: The variables are prefixed with `SHIKARI_` from `v0.3.0`. Please refer to the specific version doc to find the right variables.

> NOTE: Please open GH [issues](https://github.com/Ranjandas/shikari/issues) if you would like to have additional variables injected.

### Env

The `env` command prints various Nomad and Consul environment variables that helps you interact with the Nomad and Consul Clusters from the Host (using client binaries).

```
$ shikari env -n murphy --tls --acl consul
export CONSUL_HTTP_ADDR=https://192.168.105.13:8501
export CONSUL_HTTP_TOKEN=root

$ shikari env -n murphy --tls --acl nomad
export NOMAD_ADDR=https://192.168.105.13:4646
export NOMAD_TOKEN=00000000-0000-0000-0000-000000000000
```

The supported products are `consul`, `nomad`, `vault`, `boundary` and `k3s`, and several can be passed separated by commas (eg: `consul,nomad`). For `k3s`, the kubeconfig is copied from `srv-01` and `KUBECONFIG` is set to it.

| Flag | Effect |
|---|---|
| `-t`, `--tls` | Use HTTPS and set the CA cert variable (eg: `CONSUL_CACERT`) |
| `-a`, `--acl` | Set the bootstrap token variable (Consul and Nomad) |
| `-i`, `--insecure` | Skip TLS verification (eg: `CONSUL_HTTP_SSL_VERIFY=false`) |
| `-u`, `--unset` | Print `unset` commands for the variables instead of `export` |
| `-s`, `--server` | Point at this server instead of the first running one |

The variables point at the first running server. Use `--server` (`-s`) to pick another one, eg: `shikari env -n murphy -s srv-02 nomad`.

Use `eval` to set these environment variables in the current shell session.

```
$ eval $(shikari env -n murphy consul)

$ consul members
Node                Address              Status  Type    Build   Protocol  DC      Partition  Segment
lima-murphy-srv-01  192.168.105.13:8301  alive   server  1.18.2  2         murphy  default    <all>
lima-murphy-cli-01  192.168.105.10:8301  alive   client  1.18.2  2         murphy  default    <default>
```

### Stop

The `stop` command stops all the VMs in a cluster to save resources.

```
$ shikari stop -n <cluster-name>
```

### Start

The `start` command starts all the VMs in a stopped cluster.

```
$ shikari start -n <cluster-name>
```

> NOTE: `create`, `scale`, `start`, `stop` and `destroy` work on up to 3 VMs at a time. Set `SHIKARI_PARALLELISM` to change that, eg: `SHIKARI_PARALLELISM=1` handles one VM after another. A `start` or a delete that fails is retried once.

### Shell

> Introduced in v0.6.0

The `shell` command gets you a shell inside the VM

```
$ shikari shell <vm-name>
```

### Exec

The `exec` command takes a command as argument and executes the command against a set of servers and returns the results. You can use the following flags to filter the VMs against which the commands are executed.

| Flag | Target |
|---|---|
| `-a` | Targets all VMs in a given cluster |
| `-s` | Runs only against the `server` VMs |
| `-c` | Runs only against the `client` VMs |
| `-i <instance name>` | Targets a specific instance by its name (eg: `srv-01` or `cli-02`) |

Pass `-q` to print only the output of the commands, without the `Running command against` headers.

One of the flags is required. A single argument is run by a shell inside the VM, so it can use pipes and variables (eg: `'cat /etc/hosts | grep srv'`). Multiple arguments are run as-is, keeping their quoting (eg: `-- sh -c 'echo $HOSTNAME'`). `exec` exits with the worst exit status of the remote commands.

```
$ shikari exec -n murphy -s sudo systemctl is-enabled consul

Running command against: murphy-srv-01

enabled

Running command against: murphy-srv-02

enabled

Running command against: murphy-srv-03

enabled
```

### Scale

The `scale` command adds or removes VMs in an existing cluster. Pass the number of servers (`-s`) and/or clients (`-c`) the cluster should have; a role that isn't passed keeps its current count.

```
$ shikari scale -n murphy -c 5
```

New VMs take the lowest free numbers (eg: `cli-02` if it was removed earlier), and are created with the cluster's template and architecture unless `-t` or `-a` is passed. `-e` and `-i` work as they do for `create`.

Scaling down removes the highest numbered VMs, and needs `-f` to confirm. Scaling both roles to 0 is refused, use `destroy` instead.

```
$ shikari scale -n murphy -s 1 -f
```

### Destroy

The `destroy` command destroys the cluster as long as all the VMs in the cluster are stopped. If you want to force destroy use the `-f` flag.

```
$ shikari destroy -f -n murphy
```

### License

#### List

Lists licenses available for auto-loading and the environment variable they may populate.

## License Auto-Loading

License files (`*.hclic`) found in `~/.shikari` will be read and used to automatically populate license environment variables.

Files should be named for the product they're for, i.e. `~/.shikari/consul.hclic`. This file would be used to populate the `CONSUL_LICENSE` environment variable.

If a license environment variable is specified manually, e.g. `--env CONSUL_LICENSE=$(cat scenario_specific_consul_license.hclic)`, auto-loading will be skipped for that product.

> NOTE: Licenses are passed to the VMs as environment variables, so Lima stores their contents in plain text in each VM's `~/.lima/<vm-name>/lima.yaml`, and Lima also adds them to `/etc/environment` inside the VMs, which every user in the VM can read. They are not passed on the command line, so they don't show up in `ps`. Destroying the cluster removes the copies in `~/.lima`. Don't share a VM's `lima.yaml` or `~/.lima` directory with a license loaded.

## Feedback

Please share your feedback by creating opening a thread in the Shikari [Discussions](https://github.com/Ranjandas/shikari/discussions).
