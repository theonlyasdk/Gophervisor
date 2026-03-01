package qemu

import (
	"context"
	"fmt"
	"os/exec"
)

// ImgCreateOptions holds parameters for qemu-img create.
type ImgCreateOptions struct {
	Format        string
	Options       string
	BackingFile   string
	BackingFormat string
	Object        string
	File          string
	Size          string
}

// CreateImage executes qemu-img create.
func CreateImage(ctx context.Context, opts *ImgCreateOptions) error {
	var args []string
	args = append(args, "create")

	if opts.Format != "" {
		args = append(args, "-f", opts.Format)
	}
	if opts.Options != "" {
		args = append(args, "-o", opts.Options)
	}
	if opts.BackingFile != "" {
		args = append(args, "-b", opts.BackingFile)
	}
	if opts.BackingFormat != "" {
		args = append(args, "-B", opts.BackingFormat)
	}
	if opts.Object != "" {
		args = append(args, "--object", opts.Object)
	}
	if opts.File == "" {
		return fmt.Errorf("file path is required")
	}
	args = append(args, opts.File)

	if opts.Size != "" {
		args = append(args, opts.Size)
	} else if opts.BackingFile == "" {
		return fmt.Errorf("size is required unless backing file is specified")
	}

	cmd := exec.CommandContext(ctx, "qemu-img", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img error: %w\n%s", err, string(out))
	}

	return nil
}
