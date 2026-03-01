package qemu

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"Gophervisor/config"
)

// CaptureBuffer is a concurrency-safe in-memory output buffer.
type CaptureBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *CaptureBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *CaptureBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func buildArgs(opts *config.Options) []string {
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
	netdev := strings.TrimSpace(opts.Netdev)
	if netdev != "" && !strings.EqualFold(netdev, "none") {
		args = append(args, "-netdev", netdev)
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

	return args
}

func commandForOptions(ctx context.Context, opts *config.Options, qemuBinary string) *exec.Cmd {
	if qemuBinary == "" {
		qemuBinary = "qemu-system-x86_64"
	}
	return exec.CommandContext(ctx, qemuBinary, buildArgs(opts)...)
}

// CommandLineForOptions returns the executable and args that will be used for QEMU.
func CommandLineForOptions(opts *config.Options, qemuBinary string) []string {
	if qemuBinary == "" {
		qemuBinary = "qemu-system-x86_64"
	}
	return append([]string{qemuBinary}, buildArgs(opts)...)
}

// Start launches QEMU and returns the started command.
func Start(ctx context.Context, opts *config.Options, qemuBinary string) (*exec.Cmd, error) {
	cmd := commandForOptions(ctx, opts, qemuBinary)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("QEMU execution error: %w", err)
	}
	return cmd, nil
}

// StartWithOutput launches QEMU and captures combined stdout/stderr output.
func StartWithOutput(ctx context.Context, opts *config.Options, qemuBinary string) (*exec.Cmd, *CaptureBuffer, error) {
	cmd := commandForOptions(ctx, opts, qemuBinary)
	out := &CaptureBuffer{}
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("QEMU execution error: %w", err)
	}
	return cmd, out, nil
}

// StartDetached launches QEMU without attaching stdio so it can keep running independently.
func StartDetached(opts *config.Options, qemuBinary string) (*exec.Cmd, error) {
	if qemuBinary == "" {
		qemuBinary = "qemu-system-x86_64"
	}
	cmd := exec.Command(qemuBinary, buildArgs(opts)...)
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", os.DevNull, err)
	}
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	if err := cmd.Start(); err != nil {
		_ = devNull.Close()
		return nil, fmt.Errorf("QEMU execution error: %w", err)
	}
	_ = devNull.Close()
	return cmd, nil
}

// Run launches the QEMU emulator with the provided options.
func Run(ctx context.Context, opts *config.Options, qemuBinary string) <-chan error {
	errCh := make(chan error, 1)
	cmd := commandForOptions(ctx, opts, qemuBinary)

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
