package driver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/luthermonson/go-proxmox"
	"github.com/rancher/machine/libmachine/log"
	yaml "gopkg.in/yaml.v3"
)

// Configures cloud-init for the current machine.
func (d *Driver) setupCloudinit(ctx context.Context) error {
	machine, err := d.getCurrentMachine(ctx)
	if err != nil {
		return err
	}

	cloudinitMetadata, err := d.generateCloudinitMetadata()
	if err != nil {
		return fmt.Errorf("failed to generate cloud-init metadata: %w", err)
	}

	cloudinitUserdata, err := d.generateCloudinitUserdata(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate cloud-init userdata: %w", err)
	}

	cloudinitNetworkConfig, err := d.generateCloudinitNetworkConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate cloud-init network config: %w", err)
	}

	if err := machine.CloudInit(ctx, d.ISODeviceName, cloudinitUserdata, cloudinitMetadata, cloudinitNetworkConfig, ""); err != nil {
		return fmt.Errorf("failed to configure cloud-init for Proxmox VE virtual machine ID='%d': %w", machine.VMID, err)
	}

	return nil
}

// Blocks until cloud-init finishes setup on the current machine.
func (d *Driver) waitForCloudinit() error {
	ctx, cancel := context.WithTimeout(context.TODO(), pveTaskPollingTimeout)
	defer cancel()

	for {
		err := d.runCommandOnCurrentMachine("sudo cloud-init status --wait")
		if err == nil {
			return nil
		}

		if errors.Is(err, ErrNonZeroExitCode) {
			return fmt.Errorf("cloud-init finished with non-zero exit code: %w", err)
		}

		log.Warn("failed to execute 'sudo cloud-init status --wait' over SSH, will retry:", err.Error())

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for cloud-init to finish: %w", context.DeadlineExceeded)
		case <-time.After(pveTaskPollingInterval):
			continue
		}
	}
}

// Removes cloud-init configuration from the current machine.
func (d *Driver) cleanupCloudinit(ctx context.Context) error {
	machine, err := d.getCurrentMachine(ctx)
	if err != nil {
		return err
	}

	if err := machine.UnmountCloudInitISO(ctx, d.ISODeviceName); err != nil {
		return fmt.Errorf("failed to remove cloud-init ISO: %w", err)
	}

	err = d.runTaskOnCurrentMachine(ctx, func(ctx context.Context, vm *proxmox.VirtualMachine) (*proxmox.Task, error) {
		return vm.RemoveTag(ctx, proxmox.MakeTag(proxmox.TagCloudInit))
	})
	if err != nil {
		return fmt.Errorf("failed to remove cloud-init tag: %w", err)
	}

	return nil
}

// Generates cloud-init metadatadata for the current machine.
func (d *Driver) generateCloudinitMetadata() (string, error) {
	metadata := map[string]interface{}{
		"instance-id": d.MachineName,
		"hostname":    d.MachineName,
	}

	metadataYAML, err := yaml.Marshal(&metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cloud-init metadata: %w", err)
	}

	return string(metadataYAML), nil
}

// Generates cloud-init userdata for the current machine.
func (d *Driver) generateCloudinitUserdata(ctx context.Context) (string, error) {
	sshPublicKey, err := os.ReadFile(d.GetSSHPublicKeyPath())
	if err != nil {
		return "", fmt.Errorf("failed to read machine's SSH public key: %w", err)
	}

	userdata := map[string]interface{}{
		"hostname":             d.MachineName,
		"preserve_hostname":    false,
		"create_hostname_file": true,
		"users": []map[string]interface{}{
			{
				"name":        d.SSHUser,
				"lock_passwd": true,
				"sudo":        "ALL=(ALL) NOPASSWD:ALL",
				"ssh_authorized_keys": []string{
					string(sshPublicKey),
				},
			},
		},
	}

	// Add second NIC configuration if requested
	if d.NetworkBridge1 != "" {
		log.Debug("Adding second NIC configuration to cloud-init userdata")

		// Get current VM to read network interface MAC addresses
		machine, err := d.getCurrentMachine(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get current machine: %w", err)
		}

		// Get network interfaces from the VM
		nets := machine.VirtualMachineConfig.MergeNets()

		// Extract MAC address for net1 (second NIC)
		net1MAC, err := extractMACAddress(nets["net1"])
		if err != nil {
			return "", fmt.Errorf("failed to extract MAC address for net1: %w", err)
		}

		log.Debugf("Configuring second NIC with MAC: %s", net1MAC)

		// Add netplan config for second NIC
		userdata["write_files"] = []map[string]interface{}{
			{
				"path": "/etc/netplan/60-second-nic.yaml",
				"content": fmt.Sprintf(`network:
  version: 2
  ethernets:
    second-nic:
      match:
        macaddress: %s
      dhcp4: true
`, net1MAC),
				"permissions": "0644",
			},
		}

		userdata["runcmd"] = []string{
			"netplan apply",
		}
	}

	userdataYAML, err := yaml.Marshal(&userdata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cloud-init userdata: %w", err)
	}

	return fmt.Sprintf("#cloud-config\n%s", userdataYAML), nil
}

// Extracts MAC address from Proxmox network interface configuration string.
// Example input: "virtio=BC:24:11:2D:7A:F5,bridge=vmbr0,tag=12"
// Returns: "bc:24:11:2d:7a:f5" (lowercase as required by cloud-init)
func extractMACAddress(netConfig string) (string, error) {
	if netConfig == "" {
		return "", fmt.Errorf("network configuration is empty")
	}

	// Parse the network config string (format: "model=MAC,key=value,...")
	parts := strings.Split(netConfig, ",")
	for _, part := range parts {
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				// The MAC address is the value part of model=MAC
				// Common models: virtio, e1000, etc.
				mac := kv[1]
				// Check if this looks like a MAC address (contains colons)
				if strings.Contains(mac, ":") {
					// cloud-init expects lowercase MAC addresses
					return strings.ToLower(mac), nil
				}
			}
		}
	}

	return "", fmt.Errorf("no MAC address found in network config: %s", netConfig)
}

// Generates cloud-init network configuration for the current machine.
// Returns empty string since we handle network config via userdata instead.
func (d *Driver) generateCloudinitNetworkConfig(ctx context.Context) (string, error) {
	// We don't use network-config anymore because it overrides the template's
	// network configuration. Instead, we add the second NIC config via
	// write_files and runcmd in the userdata (see generateCloudinitUserdata).
	return "", nil
}
