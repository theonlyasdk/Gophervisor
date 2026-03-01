package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"Gophervisor/qemu"
)

func showCreateDiskDialog(parent fyne.Window, onCreated func(string)) {
	dWin := fyne.CurrentApp().NewWindow("Create Hard Disk Image")
	dWin.Resize(fyne.NewSize(500, 400))

	opts := &qemu.ImgCreateOptions{
		Format: "qcow2",
	}

	formatSelect := widget.NewSelect([]string{"qcow2", "raw", "vmdk", "vdi", "vhdx", "vpc", "qed", "parallels"}, func(s string) {
		opts.Format = s
	})
	formatSelect.SetSelected(opts.Format)

	fileEntry := widget.NewEntry()
	fileEntry.PlaceHolder = "/path/to/new-image.img"
	fileEntry.OnChanged = func(s string) {
		opts.File = s
	}

	fileBtn := widget.NewButton("Choose", func() {
		dlg := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if uc != nil && err == nil {
				path := uc.URI().Path()
				uc.Close()
				os.Remove(path) // Remove the 0-byte file immediately written by Fyne

				if filepath.Ext(path) == "" {
					path += "." + opts.Format
				}

				fileEntry.SetText(path)
				opts.File = path
			}
		}, dWin)
		dlg.Show()
	})
	fileBox := container.NewBorder(nil, nil, nil, fileBtn, fileEntry)

	sizeEntry := widget.NewEntry()
	sizeEntry.PlaceHolder = "e.g., 10G, 500M"
	sizeEntry.OnChanged = func(s string) {
		opts.Size = s
	}

	backingEntry := widget.NewEntry()
	backingEntry.PlaceHolder = "/path/to/backing-image.img"
	backingEntry.OnChanged = func(s string) {
		opts.BackingFile = s
	}

	backingBtn := widget.NewButton("Choose", func() {
		dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
			if uc != nil && err == nil {
				backingEntry.SetText(uc.URI().Path())
				opts.BackingFile = uc.URI().Path()
			}
		}, dWin)
	})
	backingBox := container.NewBorder(nil, nil, nil, backingBtn, backingEntry)

	backingFmtSelect := widget.NewSelect([]string{"qcow2", "raw", "vmdk", "vdi", "vhdx"}, func(s string) {
		opts.BackingFormat = s
	})
	backingFmtSelect.PlaceHolder = "Auto"

	optionsEntry := widget.NewEntry()
	optionsEntry.PlaceHolder = "e.g., clustering_size=128k"
	optionsEntry.OnChanged = func(s string) {
		opts.Options = s
	}

	form := widget.NewForm(
		&widget.FormItem{Text: "Format (-f)", Widget: formatSelect, HintText: "Disk image format (default: qcow2)"},
		&widget.FormItem{Text: "File", Widget: fileBox, HintText: "Target file path"},
		&widget.FormItem{Text: "Size", Widget: sizeEntry, HintText: "Image size with multiplier (e.g., 10G, 500M)"},
		&widget.FormItem{Text: "Backing File (-b)", Widget: backingBox, HintText: "CoW base image"},
		&widget.FormItem{Text: "Backing Format (-B)", Widget: backingFmtSelect, HintText: "Format of the backing file"},
		&widget.FormItem{Text: "Format Options (-o)", Widget: optionsEntry, HintText: "Format-specific opts (e.g. preallocation=full)"},
	)

	createBtn := widget.NewButton("Create Image", func() {
		if opts.File == "" {
			dialog.ShowError(fmt.Errorf("File path is required"), dWin)
			return
		}
		if opts.Size == "" && opts.BackingFile == "" {
			dialog.ShowError(fmt.Errorf("Size is required unless a backing file is specified"), dWin)
			return
		}
		err := qemu.CreateImage(context.Background(), opts, qemuImgBinaryPath())
		if err != nil {
			dialog.ShowError(err, dWin)
		} else {
			dialog.ShowInformation("Success", "Disk image space initialized: "+opts.File, dWin)
			if onCreated != nil {
				onCreated(opts.File)
			}
		}
	})

	content := container.NewPadded(container.NewBorder(nil, container.NewPadded(createBtn), nil, nil, container.NewVScroll(form)))
	dWin.SetContent(content)
	dWin.Show()
}
