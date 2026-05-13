# Changelog

## [2.1.0] - 2026-05-13

### Added

- Add preflight checks during `keights quickstart` and new command `keights account validate`.

### Changed

- Update CI builds to run on pushes for any branch.
- Identify nodes by EC2 instance ID instead of hostname.
- Update easyto to v0.11.0.

### Fixed

- Fix AMI version and owner resolution.

## [2.0.0] - 2026-04-27

This is a rewrite from the previous Ansible and CloudFormation based version.

Differences include:

- AMIs are now built with https://github.com/cloudboss/easyto, which brings nodes up more quickly.
- Configured to use AWS VPC CNI plugin, while the original used either Kube Router or Calico.
- Configured out of the box for aws-iam-authenticator, which enables more secure cluster access and node bootstrapping. Gone are long lived auth certificates and bootstrap tokens.
- Nodes use faster, simpler bootstrapping rather than running kubeadm.
- Uses a network load balancer for the API while the original used a classic ELB.
- Uses AWS cloud controller manager rather than the deprecated builtin controller.
- Includes [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) support by default.
- Includes a CLI tool to manage clusters with quickstart and other utilities.
- Keights version is decoupled from Kubernetes versions so one keights release can support multiple Kubernetes releases.

[2.1.0]: https://github.com/cloudboss/keights/releases/tag/v2.1.0
[2.0.0]: https://github.com/cloudboss/keights/releases/tag/v2.0.0
