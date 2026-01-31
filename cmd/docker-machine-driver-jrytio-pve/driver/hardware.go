package driver

import (
	"context"
	"fmt"
	"strings"

	"github.com/luthermonson/go-proxmox"
	"github.com/rancher/machine/libmachine/log"
)

// Configures hardware for the current machine.
func (d *Driver) setupHardware(ctx context.Context) error {
	options := make([]proxmox.VirtualMachineOption, 0)

	if d.ProcessorSockets != nil {
		options = append(options, proxmox.VirtualMachineOption{
			Name:  "sockets",
			Value: *d.ProcessorSockets,
		})
	}

	if d.ProcessorCores != nil {
		options = append(options, proxmox.VirtualMachineOption{
			Name:  "cores",
			Value: *d.ProcessorCores,
		})
	}

	if d.Memory != nil {
		options = append(options, proxmox.VirtualMachineOption{
			Name:  "memory",
			Value: *d.Memory,
		})
	}

	if d.MemoryBalloon != nil {
		options = append(options, proxmox.VirtualMachineOption{
			Name:  "balloon",
			Value: *d.MemoryBalloon,
		})
	}

	// Configure network adapters if requested
	if d.ConfigureNetwork {
		networkOptions, err := d.buildNetworkOptions(ctx)
		if err != nil {
			return fmt.Errorf("failed to build network configuration: %w", err)
		}
		options = append(options, networkOptions...)
	}

	if len(options) < 1 {
		return nil
	}

	err := d.runTaskOnCurrentMachine(ctx, func(ctx context.Context, vm *proxmox.VirtualMachine) (*proxmox.Task, error) {
		return vm.Config(ctx, options...)
	})
	if err != nil {
		return fmt.Errorf("failed to configure hardware: %w", err)
	}

	return nil
}

// Builds network configuration options for the current machine.
func (d *Driver) buildNetworkOptions(ctx context.Context) ([]proxmox.VirtualMachineOption, error) {
	options := make([]proxmox.VirtualMachineOption, 0)

	// Get current VM config to read existing network settings
	machine, err := d.getCurrentMachine(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current machine: %w", err)
	}

	// Get existing network interfaces from the VM
	nets := machine.VirtualMachineConfig.MergeNets()

	// Configure net0 if bridge or VLAN is specified
	if d.NetworkBridge0 != "" || d.NetworkVlan0 != "" {
		net0Config, err := d.buildNetworkInterfaceConfig("net0", nets, d.NetworkBridge0, d.NetworkVlan0)
		if err != nil {
			return nil, fmt.Errorf("failed to build net0 configuration: %w", err)
		}
		log.Debugf("Configuring net0: %s", net0Config)
		options = append(options, proxmox.VirtualMachineOption{
			Name:  "net0",
			Value: net0Config,
		})
	}

	// Configure net1 if bridge is specified
	if d.NetworkBridge1 != "" {
		net1Config, err := d.buildNetworkInterfaceConfig("net1", nets, d.NetworkBridge1, d.NetworkVlan1)
		if err != nil {
			return nil, fmt.Errorf("failed to build net1 configuration: %w", err)
		}
		log.Debugf("Configuring net1: %s", net1Config)
		options = append(options, proxmox.VirtualMachineOption{
			Name:  "net1",
			Value: net1Config,
		})
	}

	return options, nil
}

// Builds a network interface configuration string.
func (d *Driver) buildNetworkInterfaceConfig(interfaceName string, existingNets map[string]string, bridge, vlan string) (string, error) {
	// Start with existing configuration if available, otherwise default to virtio
	existingConfig, exists := existingNets[interfaceName]

	var model string
	var mac string

	if exists && existingConfig != "" {
		// Parse existing configuration to preserve model and MAC
		parts := strings.Split(existingConfig, ",")
		for _, part := range parts {
			if strings.Contains(part, "=") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) == 2 {
					key := kv[0]
					value := kv[1]

					// Check if this is the model/MAC part
					models := []string{"virtio", "e1000", "e1000-82540em", "e1000-82544gc", "e1000-82545em",
						"e1000e", "i82551", "i82557b", "i82559er", "ne2k_isa", "ne2k_pci", "pcnet", "rtl8139", "vmxnet3"}
					for _, m := range models {
						if key == m {
							model = m
							mac = value
							break
						}
					}
				}
			} else {
				// Could be a standalone model name
				if model == "" && part != "" {
					model = part
				}
			}
		}
	}

	// Default to virtio if no model found
	if model == "" {
		model = "virtio"
	}

	// Build the configuration string
	var configParts []string

	if mac != "" {
		configParts = append(configParts, fmt.Sprintf("%s=%s", model, mac))
	} else {
		configParts = append(configParts, model)
	}

	// Add bridge if specified
	if bridge != "" {
		configParts = append(configParts, fmt.Sprintf("bridge=%s", bridge))
	}

	// Add VLAN tag if specified
	if vlan != "" {
		configParts = append(configParts, fmt.Sprintf("tag=%s", vlan))
	}

	return strings.Join(configParts, ","), nil
}
