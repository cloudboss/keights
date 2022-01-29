packer {
  required_plugins {
    amazon = {
      version = "= 1.0.8"
      source = "github.com/hashicorp/amazon"
    }
  }
}

variable "ami-architecture" {
  type    = string
  default = "x86_64"
}

variable "ami-name-suffix" {
  type    = string
  default = ""
}

variable "base-ami-owner" {
  type    = string
  default = "379101102735"
}

variable "base-ami-pattern" {
  type    = string
  default = "debian-stretch-hvm-x86_64-gp2-*"
}

variable "containerd-version" {
  type    = string
  default = "1.4.9-1"
}

variable "debian-release" {
  type    = string
  default = "bullseye"
}

variable "dev-mode" {
  type    = bool
  default = false
}

variable "k8s-version" {
  type    = string
  default = ""
}

variable "keights-version" {
  type    = string
  default = ""
}

variable "root-vol" {
  type    = string
  default = "xvdb"
}

variable "share-accounts" {
  type    = string
  default = "all"
}

variable "ssh-interface" {
  type    = string
  default = "public_ip"
}

variable "subnet-id" {
  type    = string
  default = ""
}

variable "usr-vol" {
  type    = string
  default = "xvdc"
}

variable "var-lib-containerd-vol" {
  type    = string
  default = "xvde"
}

variable "var-log-vol" {
  type    = string
  default = "xvdf"
}

variable "var-vol" {
  type    = string
  default = "xvdd"
}

variable "vpc-id" {
  type    = string
  default = ""
}

data "amazon-ami" "base_ami" {
  filters = {
    name                = var.base-ami-pattern
    root-device-type    = "ebs"
    virtualization-type = "hvm"
  }
  most_recent = true
  owners      = [var.base-ami-owner]
}

source "amazon-ebssurrogate" "base_ami" {
  ami_description = "Cloudboss Keights Debian ${var.debian-release}"
  ami_groups      = [var.share-accounts]
  ami_name        = "debian-${var.debian-release}-k8s-hvm-amd64-${var.keights-version}${var.ami-name-suffix}"
  ami_root_device {
    delete_on_termination = true
    device_name           = "/dev/xvda"
    source_device_name    = "/dev/${var.root-vol}"
    volume_size           = 1
    volume_type           = "gp2"
  }
  ami_architecture            = var.ami-architecture
  ami_virtualization_type     = "hvm"
  boot_mode                   = "uefi"
  associate_public_ip_address = true
  ena_support                 = true
  instance_type               = "t2.micro"
  launch_block_device_mappings {
    delete_on_termination = true
    device_name           = "/dev/${var.root-vol}"
    volume_size           = 1
    volume_type           = "gp2"
  }
  launch_block_device_mappings {
    delete_on_termination = true
    device_name           = "/dev/${var.usr-vol}"
    volume_size           = 2
    volume_type           = "gp2"
  }
  launch_block_device_mappings {
    delete_on_termination = true
    device_name           = "/dev/${var.var-vol}"
    volume_size           = 1
    volume_type           = "gp2"
  }
  launch_block_device_mappings {
    delete_on_termination = true
    device_name           = "/dev/${var.var-lib-containerd-vol}"
    volume_size           = 22
    volume_type           = "gp2"
  }
  launch_block_device_mappings {
    delete_on_termination = true
    device_name           = "/dev/${var.var-log-vol}"
    volume_size           = 8
    volume_type           = "gp2"
  }
  run_tags = {
    Name = "ami-builder-${var.debian-release}"
  }
  run_volume_tags = {
    Name = "ami-volume-${var.debian-release}"
  }
  snapshot_groups = [var.share-accounts]
  source_ami      = "${data.amazon-ami.base_ami.id}"
  sriov_support   = true
  ssh_interface   = var.ssh-interface
  ssh_pty         = true
  ssh_timeout     = "5m"
  ssh_username    = "admin"
  subnet_id       = var.subnet-id
  tags = {
    "containerd:version" = var.containerd-version
    "k8s:version"        = var.k8s-version
    "keights:version"    = var.keights-version
    "os:version"         = "debian-${var.debian-release}"
  }
  vpc_id = var.vpc-id
}

build {
  sources = ["source.amazon-ebssurrogate.base_ami"]

  provisioner "ansible" {
    extra_arguments     = [
      "-e", "root_vol=${var.root-vol}",
      "-e", "usr_vol=${var.usr-vol}",
      "-e", "var_vol=${var.var-vol}",
      "-e", "var_lib_containerd_vol=${var.var-lib-containerd-vol}",
      "-e", "var_log_vol=${var.var-log-vol}",
      "-e", "containerd_version=${var.containerd-version}",
      "-e", "k8s_version=${var.k8s-version}",
      "-e", "keights_version=${var.keights-version}",
      "-e", "debian_release=${var.debian-release}",
      "-e", "dev_mode=${var.dev-mode}"
    ]
    keep_inventory_file = var.dev-mode
    playbook_file       = "./playbook.yml"
    use_proxy           = false
    user                = "admin"
  }
}
