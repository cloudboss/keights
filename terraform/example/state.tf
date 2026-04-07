terraform {
  backend "s3" {
    bucket       = "terraform-state-f34ehzkvbv"
    key          = "keights/bonito/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = true
  }
}
