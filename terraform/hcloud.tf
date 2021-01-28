terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "1.24.0"
    }
  }
}


# Set the variable value in *.tfvars file
# or using the -var="hcloud_token=..." CLI option
variable "hcloud_token" {}
variable "server_name" {}
variable "server_sshkey" {}


# Configure the Hetzner Cloud Provider
provider "hcloud" {
  token = var.hcloud_token
}

# Create a production server
resource "hcloud_server" "server" {
  name     = var.server_name
  image    = "ubuntu-20.04"
  location = "fsn1"
  # 1vCPU, 2GB RAM, 20GB NVMe SSD, 20TB Traffic
  server_type = "cx11"

  # reference cloud-init.yaml file
  user_data = templatefile(
    "${path.module}/cloud-init.yaml",
    {
      username = "ubuntu"
      sshkey   = "${var.server_sshkey}"
    }
  )
}

# Output
output "server_ipv4" {
  value = hcloud_server.server.ipv4_address
}

output "server_ipv6" {
  value = hcloud_server.server.ipv6_address
}
