packer {
  required_plugins {
    proxmox = {
      version = "~> 1.2.2"
      source  = "github.com/hashicorp/proxmox"
    }
  }
}

variable "pve_url" {
  type = string
}

variable "pve_insecure_tls" {
  type    = bool
  default = false
}

variable "pve_token_id" {
  type = string
}

variable "pve_token_secret" {
  type      = string
  sensitive = true
}

variable "pve_node" {
  type = string
}

variable "pve_pool" {
  type = string
}

variable "pve_id" {
  type    = number
  default = null
}

variable "pve_name" {
  type    = string
  default = "ubuntu-server-24.04-cloudinit"
}

variable "pve_storage_iso" {
  type = string
}

variable "pve_storage_disk" {
  type = string
}

variable "pve_network_bridge" {
  type = string
}

variable "ssh_username" {
  type = string
}

variable "ssh_authorized_key" {
  type = string
}

source "proxmox-iso" "ubuntu" {
  proxmox_url              = "${var.pve_url}/api2/json"
  insecure_skip_tls_verify = "${var.pve_insecure_tls}"
  username                 = "${var.pve_token_id}"
  token                    = "${var.pve_token_secret}"

  node         = "${var.pve_node}"
  pool         = "${var.pve_pool}"
  task_timeout = "5m"

  vm_id                = "${var.pve_id}"
  template_name        = "${var.pve_name}"
  template_description = "Packer config: https://github.com/stellatarum/docker-machine-driver-pve/blob/main/deploy/templates/ubuntu-server"
  os                   = "l26"
  machine              = "q35"
  qemu_agent           = true
  cloud_init           = false
  onboot               = false

  cpu_type           = "x86-64-v2-AES"
  sockets            = 1
  cores              = 8
  memory             = 8192
  ballooning_minimum = 0

  scsi_controller = "virtio-scsi-single"
  disks {
    type         = "scsi"
    storage_pool = "${var.pve_storage_disk}"
    disk_size    = "20G"
    format       = "raw"
    io_thread    = true
    ssd          = true
    discard      = true
    cache_mode   = "none"
  }

  network_adapters {
    model    = "virtio"
    bridge   = "${var.pve_network_bridge}"
    firewall = "false"
  }

  network_adapters {
    model    = "virtio"
    bridge   = "${var.pve_network_bridge}"
    firewall = "false"
  }

  boot_iso {
    type  = "scsi"
    index = 1

    iso_url           = "https://releases.ubuntu.com/noble/ubuntu-24.04.3-live-server-amd64.iso"
    iso_checksum      = "sha256:c3514bf0056180d09376462a7a1b4f213c1d6e8ea67fae5c25099c6fd3d8274b"
    iso_storage_pool  = "${var.pve_storage_iso}"
    iso_download_pve  = true
    unmount           = true
    keep_cdrom_device = true
  }

  http_content = {
    "/meta-data" = templatefile("${path.root}/cloud-init/meta-data.pkrtpl.hcl", {})
    "/user-data" = templatefile("${path.root}/cloud-init/user-data.pkrtpl.hcl", {
      hostname           = "${var.pve_name}"
      ssh_username       = "${var.ssh_username}"
      ssh_authorized_key = "${var.ssh_authorized_key}"
    })
  }

  boot_command = [
    "<esc><wait>",
    "e<wait>",
    "<down><down><down><end>",
    "<bs><bs><bs><bs><wait>",
    "autoinstall ds=nocloud-net\\;s=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ ---<wait>",
    "<f10><wait>"
  ]
  boot      = "order=scsi0;scsi1"
  boot_wait = "5s"

  ssh_agent_auth = true
  ssh_username   = "${var.ssh_username}"
  ssh_timeout    = "10m"
}

build {
  name    = "ubuntu-server-24.04-cloudinit"
  sources = ["source.proxmox-iso.ubuntu"]

  provisioner "shell" {
    inline = [
      "sudo cloud-init status --wait",
      "sudo cloud-init clean --machine-id",
      "sudo rm /etc/ssh/ssh_host_*key*",
    ]
  }
}
