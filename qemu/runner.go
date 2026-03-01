package qemu

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"Gophervisor/config"
)

// Run launches the QEMU emulator with the provided options.
func Run(ctx context.Context, opts *config.Options) <-chan error {
	errCh := make(chan error, 1)

	var args []string

	if opts.Machine != "" {
		args = append(args, "-machine", opts.Machine)
	}
	if opts.Accel != "" {
		args = append(args, "-accel", opts.Accel)
	}
	if opts.CPU != "" {
		args = append(args, "-cpu", opts.CPU)
	}
	if opts.SMP != "" {
		args = append(args, "-smp", opts.SMP)
	}
	if opts.Memory != "" {
		args = append(args, "-m", opts.Memory)
	}
	if opts.Boot != "" {
		args = append(args, "-boot", opts.Boot)
	}
	if opts.DriveHDA != "" {
		args = append(args, "-hda", opts.DriveHDA)
	}
	if opts.DriveCDROM != "" {
		args = append(args, "-cdrom", opts.DriveCDROM)
	}
	if opts.VGA != "" {
		args = append(args, "-vga", opts.VGA)
	}
	if opts.Display != "" {
		args = append(args, "-display", opts.Display)
	}
	if opts.Netdev != "" {
		args = append(args, "-netdev", opts.Netdev)
	}
	if opts.Kernel != "" {
		args = append(args, "-kernel", opts.Kernel)
	}
	if opts.Initrd != "" {
		args = append(args, "-initrd", opts.Initrd)
	}
	if opts.Append != "" {
		args = append(args, "-append", opts.Append)
	}
	if opts.USB {
		args = append(args, "-usb")
	}
	if opts.USBDevice != "" {
		args = append(args, "-usbdevice", opts.USBDevice)
	}
	if opts.TPMDev != "" {
		args = append(args, "-tpmdev", opts.TPMDev)
	}
	if opts.CharDev != "" {
		args = append(args, "-chardev", opts.CharDev)
	}
	if opts.DebugOptions != "" {
		args = append(args, strings.Fields(opts.DebugOptions)...)
	}
	if opts.GenericObj != "" {
		args = append(args, "-object", opts.GenericObj)
	}
	if opts.ExtraOptions != "" {
		args = append(args, strings.Fields(opts.ExtraOptions)...)
	}

	cmd := exec.CommandContext(ctx, "qemu-system-x86_64", args...)

	go func() {
		out, err := cmd.CombinedOutput()
		if err != nil {
			errCh <- fmt.Errorf("QEMU Execution Error: %w\nOutput: %s", err, string(out))
		} else {
			errCh <- nil
		}
		close(errCh)
	}()

	return errCh
}
