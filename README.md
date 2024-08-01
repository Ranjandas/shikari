# shikari

Shikari is an opinionated tool that helps you create multi-vm clusters (primarly aimed at Nomad and Consul) using [Lima](https://lima-vm.io) as the underlying VM Manager.

Various scenarios built for Shikari is available here: https://github.com/Ranjandas/shikari-scenarios/

## Prerequisites

The following are the pre-requisites for using Shikari

* Lima: Used for running VMs
* QEMU [Optional]: Required if you are also baking images using packer
* HashiCorp Packer: To build custom images (available in the shikar-scenarios/packer repository)
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

### List

The `list` command is used to list the clusters and their VMs. You can get VMs of a specific cluster by passing the `--name/-n` flag.

```
$ CLUSTER       VM NAME             SATUS         DISK(GB)       MEMORY(GB)       CPUS       IMAGE
murphy        murphy-cli-01       Running       100            4                4          /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy        murphy-cli-02       Running       100            4                4          /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy        murphy-cli-03       Running       100            4                4          /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy        murphy-srv-01       Running       100            4                4          /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy        murphy-srv-02       Running       100            4                4          /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
murphy        murphy-srv-03       Running       100            4                4          /Users/ranjan/workspace/github/shikari-scenarios/packer/.artifacts/c-1.18-n-1.7/c-1.18-n-1.7.qcow2
```

#### Helper Variables

When spinning up the VM's, Shikari injects a few environment variables into each VM's, which would give some additional context to the provisioning scripts that would include:

* What kind of node the script is running on (injected as `SHIKARI_VM_MODE` env variable)
* Name of the cluster (injected as `SHIKARI_CLUSTER_NAME` env variable)
* Count of Number of Servers (injected as `SHIKARI_SERVER_COUNT` env variable)
* Count of Number of Clients (injected as `SHIKARI_CLIENT_COUNT` env variable)
* Launch mode of VMs (`CREATE` or `SCALE`) (injected as `SHIKARI_LAUNCH_MODE` env variable)

> NOTE: The variables are prefixed with `SHIKARI_` from `v0.3.0`. Please refer to the specific version doc to find the right variables.

> NOTE: Please open GH [issues](https://github.com/Ranjandas/shikari/issues) if you would like to have additional variables injected.

### Env

The `env` command prints various Nomad and Consul environment variables that helps you interact with the Nomad and Consul Clusters form the Host (using client binaries).

```
$ shikari env -n murphy --tls --acl consul
export CONSUL_HTTP_ADDR=https://192.168.105.13:8501
export CONSUL_HTTP_TOKEN=root

$ shikari env -n murphy --tls --acl nomad
export NOMAD_ADDR=https://192.168.105.13:4646
export NOMAD_TOKEN=00000000-0000-0000-0000-000000000000
```

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

### Exec

The `exec` command takes a command as argument and executes the command against a set of servers and returns the results. You can use the following flags to filter the VMs against which the commands are executed.

| Flag | Target |
|---|---|
| `-a` | Targets all VMs in a given cluster |
| `-s` | Runs only against the `server` VMs |
| `-c` | Runs only against the `client` VMs |
| `-i <instance name>` | Targets a specific instance by its name (eg: `srv-01` or `cli-02`) |

```
$ shikari exec -n murphy -s sudo systemctl is-enabled consul

Running command againt: murphy-srv-01

enabled

Running command againt: murphy-srv-02

enabled

Running command againt: murphy-srv-03

enabled
```

### Destroy

The `destroy` command destroys the cluster as long as all the VMs in the cluster are stopped. If you want to force destroy use the `-f` flag.

```
$ shikari destroy -f -n murphy
```


## Feedback

Please share your feedback by creating opening a thread in the Shikari [Discussions](https://github.com/Ranjandas/shikari/discussions).