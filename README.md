# keights

An installer for self hosted Kubernetes clusters in AWS.

# Install the command

Download and unpack the latest [release](https://github.com/cloudboss/keights/releases/latest) for your platform.

Ensure it is in a directory on your `PATH` or `keights kubeconfig` and `keights kubectl` subcommands won't work.

```sh
curl -LO https://github.com/cloudboss/keights/releases/download/v2.0.0/keights-v2.0.0-darwin-amd64.tar.gz
sudo tar -C /usr/local/bin -zxf keights-v2.0.0-darwin-amd64.tar.gz keights
```

# Quickstart: create a cluster

To create your first cluster, run `keights quickstart` with your AWS credentials in scope. This will guide you through the configuration and result in a Terraform root module in your directory of choice. If you select deploy immediately, it will run Terraform in the root module directory. Otherwise, run `keights deploy <directory>` with your AWS credentials in scope. Terraform will be downloaded and cached automatically.

The quickstart does not configure every possible option, but the Terraform module can be modified according to the [documentation](./terraform/README.md).

You can also run `keights quickstart --non-interactive` to configure a cluster without an interactive wizard. See `keights quickstart --help` for all of the available options.

# EC2 AMI

The cluster depends on an official AMI built with [easyto](https://github.com/cloudboss/easyto), published in `us-east-1`. You can copy it to your own account and region using `keights ami copy --region <aws-region>`. If you are already deploying to `us-east-1`, you can use the official one without copying.

There may be multiple AMIs for a given keights version supporting different Kubernetes versions. Use `keights ami list` to see which ones are available. When you run `keights quickstart`, it should automatically choose the latest one for the currently running keights version, but you can also pass in ones of the ones listed by `keights ami list`.

# Connecting to the cluster

The first way to connect to the cluster is to run `keights kubectl --cluster-name <cluster-name> -- [kubectl args]`. Your AWS credentials must be in scope because your IAM principal is used to [authenticate](#authentication) you to the cluster.

```sh
export AWS_PROFILE=acme
keights kubectl --cluster-name bonito -- get nodes
keights kubectl --cluster-name bonito -- -n kube-system get pods
```

When you run `keights kubectl`, the `kubectl` command will be automatically downloaded and cached.

The other way to connect is to generate a [kubeconfig](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/) file and use `kubectl` directly.

```
export AWS_PROFILE=acme
keights kubeconfig --cluster-name bonito --output ~/.kube/config.
```

> [!NOTE]
> For both `keights kubectl` and `keights kubeconfig`, the `keights` command must be in a directory on your `PATH` because the kubeconfig file used by both runs the `keights token` command.

# Authentication

[AWS IAM Authenticator](https://github.com/kubernetes-sigs/aws-iam-authenticator) runs on the control plane and handles authentication via IAM. You must have your AWS credentials in scope to connect to the cluster.

By default, the principal that deployed the cluster will have admin permissions, and nodes will have bootstrap permissions.

To include additional role mappings, pass the Terraform variable [`identity_mappings`](./terraform/README.md#identity_mappings-object). Note this uses the `IAMIdentityMapping` CRDs and deploys one for each mapping.

```hcl
module "keights" {
  source = "https://github.com/cloudboss/keights/releases/download/v2.0.0/keights-terraform-v2.0.0.tar.gz"

  identity_mappings = [
    {
      name = "kubernetes-admin-session"
      spec = {
        arn      = "arn:aws:iam::000000000000:role/KubernetesAdmin"
        username = "admin:{{SessionName}}"
        groups   = ["system:masters"]
    }
  ]
```
