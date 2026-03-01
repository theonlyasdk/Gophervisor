package config

import (
	"errors"
	"strings"
)

// Options holds configuration for QEMU parameters.
type Options struct {
	Machine      string
	Accel        string
	CPU          string
	SMP          string
	Memory       string
	Boot         string
	DriveHDA     string
	DriveCDROM   string
	VGA          string
	Display      string
	VNCAddress   string
	VNCDisplay   string
	VNCPassword  bool
	VNCWebPort   string
	Netdev       string
	Kernel       string
	Initrd       string
	Append       string
	USB          bool
	USBDevice    string
	TPMDev       string
	CharDev      string
	DebugOptions string
	GenericObj   string
	ExtraOptions string
}

// NewOptions returns a default configuration.
func NewOptions() *Options {
	return &Options{
		Machine:    "q35",
		Accel:      "kvm",
		CPU:        "host",
		Memory:     "2048",
		Display:    "gtk",
		VNCAddress: "localhost",
		VNCDisplay: "1",
		VNCWebPort: "6080",
		VGA:        "virtio",
	}
}

// Validate configuration options.
func (o *Options) Validate() error {
	if o.Memory == "" {
		return errors.New("memory field is required")
	}
	if o.Machine == "" {
		return errors.New("machine field is required")
	}
	if o.CPU == "host" && o.Accel != "kvm" && o.Accel != "hvf" && o.Accel != "nvmm" && o.Accel != "whpx" {
		return errors.New("CPU model 'host' requires a hardware accelerator (e.g., kvm, hvf)")
	}
	if o.Kernel == "" && o.Initrd != "" {
		return errors.New("Initrd requires a Kernel image")
	}
	if o.Kernel == "" && o.Append != "" {
		return errors.New("Append line requires a Kernel image")
	}
	netdev := strings.TrimSpace(o.Netdev)
	if netdev != "" && !strings.EqualFold(netdev, "none") {
		parts := strings.Split(netdev, ",")
		hasID := false
		for _, part := range parts[1:] {
			if strings.HasPrefix(strings.TrimSpace(part), "id=") {
				hasID = true
				break
			}
		}
		if !hasID {
			return errors.New("Netdev ID is required when a network backend other than 'none' is selected")
		}
	}
	return nil
}
