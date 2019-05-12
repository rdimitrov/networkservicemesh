# Using `kind` as a cluster provider

[`kind`](kind.sigs.k8s.io) is a tool for running local Kubernetes clusters using Docker container “nodes”.
Docker is the only prerequisite, it does not require any additional steps, hypervisors etc.

It is worth noting that `kind` as any other Kubernetes deployment tool would expect that the machine that hosts the Docker has at least 4 CPU cores and 4 GB of RAM. That is specifically pointed for OSX users in the official [docs](https://kind.sigs.k8s.io/docs/user/quick-start/).


## `kind` lifecycle management

To start a `kind` cluster, just run:

```shell
$ make kind-start
Creating cluster "nsm" ...
 ✓ Ensuring node image (kindest/node:v1.13.3) 🖼
 ✓ Preparing nodes 📦📦📦
 ✓ Creating kubeadm config 📜
 ✓ Starting control-plane 🕹️
 ✓ Joining worker nodes 🚜
Cluster creation complete. You can now use the cluster with:

export KUBECONFIG="$(kind get kubeconfig-path --name="nsm")"
kubectl cluster-info
node/nsm-control-plane untainted
```

As seen on the last lines, to point your `kubectl` command to the new cluster one should run:

```shell
export KUBECONFIG="$(kind get kubeconfig-path --name="nsm")"
```

Deleting the cluster is as easy as:

```shell
$ make kind-stop
Deleting cluster "nsm" ...
$KUBECONFIG is still set to use $HOME/.kube/kind-config-nsm even though that file has been deleted, remember to unset it
```

## `kind` as a cluster provide in Network Service Mesh

Enabling `kind` instead of the default `vagrant` is as easy as:
```shell
export CLUSTER_RULES_PREFIX=kind
```

All subsequent commands will assume `kind` as the cluster provider.

## Latest `kind`

The default behaviour is to use the installed `kind` version and not update it. An update can be forces by:

```shell
make kind-install
```