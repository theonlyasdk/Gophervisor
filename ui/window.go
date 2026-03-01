package ui

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"Gophervisor/config"
	"Gophervisor/qemu"
)

// BuildAndRun constructs the GUI and starts the event loop.
func BuildAndRun() {
	a := app.NewWithID("com.gophervisor")

	// Check if QEMU is installed
	if _, err := exec.LookPath("qemu-system-x86_64"); err != nil {
		warnWin := a.NewWindow("QEMU Not Found")
		warnWin.Resize(fyne.NewSize(450, 250))
		lbl := widget.NewLabel("QEMU is not installed or not found in PATH.\n\nPlease install QEMU to use Gophervisor.\n\nUbuntu/Debian:\n    sudo apt install qemu-system-x86 qemu-utils\n\nArch Linux:\n    sudo pacman -S qemu-desktop\n\nmacOS (Homebrew):\n    brew install qemu")
		btn := widget.NewButton("Quit", func() {
			a.Quit()
		})
		warnWin.SetContent(container.NewBorder(nil, btn, nil, nil, lbl))
		warnWin.ShowAndRun()
		return
	}

	w := a.NewWindow("Gophervisor")
	w.Resize(fyne.NewSize(700, 500))

	opts := config.NewOptions()

	tabs := container.NewAppTabs(
		buildStandardTab(w, opts),
		buildBlockTab(w, opts),
		buildDisplayTab(w, opts),
		buildNetworkTab(w, opts),
		buildKernelTab(w, opts),
		buildMiscTab(w, opts),
		buildAdvancedTab(w, opts),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	refreshUI := func() {
		tabs.Items[0] = buildStandardTab(w, opts)
		tabs.Items[1] = buildBlockTab(w, opts)
		tabs.Items[2] = buildDisplayTab(w, opts)
		tabs.Items[3] = buildNetworkTab(w, opts)
		tabs.Items[4] = buildKernelTab(w, opts)
		tabs.Items[5] = buildMiscTab(w, opts)
		tabs.Items[6] = buildAdvancedTab(w, opts)
		tabs.Refresh()
	}

	menu := fyne.NewMainMenu(
		fyne.NewMenu("File",
			fyne.NewMenuItem("Load Preset...", func() {
				dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
					if r == nil || err != nil {
						return
					}
					defer r.Close()
					if err := json.NewDecoder(r).Decode(opts); err != nil {
						dialog.ShowError(err, w)
					} else {
						refreshUI()
					}
				}, w)
			}),
			fyne.NewMenuItem("Save Preset...", func() {
				dlg := dialog.NewFileSave(func(wr fyne.URIWriteCloser, err error) {
					if wr == nil || err != nil {
						return
					}
					
					path := wr.URI().Path()
					if filepath.Ext(path) == "" {
						wr.Close()
						os.Remove(path)
						path += ".json"
						
						f, err := os.Create(path)
						if err != nil {
							dialog.ShowError(err, w)
							return
						}
						defer f.Close()
						enc := json.NewEncoder(f)
						enc.SetIndent("", "  ")
						if err := enc.Encode(opts); err != nil {
							dialog.ShowError(err, w)
						}
						return
					}

					defer wr.Close()
					enc := json.NewEncoder(wr)
					enc.SetIndent("", "  ")
					if err := enc.Encode(opts); err != nil {
						dialog.ShowError(err, w)
					}
				}, w)
				dlg.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
				dlg.Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				a.Quit()
			}),
		),
		fyne.NewMenu("Disks",
			fyne.NewMenuItem("Create Hard Disk Image...", func() {
				showCreateDiskDialog(w, nil)
			}),
		),
		fyne.NewMenu("Help",
			fyne.NewMenuItem("About", func() {
				dialog.ShowInformation("About Gophervisor", "Gophervisor\n\nA friendly QEMU graphical frontend written in Go.", w)
			}),
		),
	)
	w.SetMainMenu(menu)

	presetsOpts := map[string]*config.Options{
		"Default Q35":         {Machine: "q35", Accel: "kvm", CPU: "host", Memory: "2048", Display: "gtk", VGA: "virtio"},
		"Legacy PC":           {Machine: "pc", Accel: "tcg", CPU: "qemu64", Memory: "1024", Display: "sdl", VGA: "std"},
		"MicroVM (Tiny)":      {Machine: "microvm", Accel: "kvm", CPU: "host", Memory: "512", Display: "none", VGA: "none", Netdev: "user"},
		"Windows 10/11 Gam":   {Machine: "q35", Accel: "kvm", CPU: "host", Memory: "8192", Display: "gtk", VGA: "virtio", GenericObj: "rng-random,filename=/dev/urandom,id=rng0"},
		"Headless Server":     {Machine: "q35", Accel: "kvm", CPU: "host", Memory: "4096", Display: "none", VGA: "none", Netdev: "user", CharDev: "stdio,id=char0"},
		"Old MS-DOS DB":       {Machine: "isapc", Accel: "tcg", CPU: "486", Memory: "16", Display: "sdl", VGA: "cirrus"},
		"Linux Kernel Debug":  {Machine: "q35", Accel: "tcg", CPU: "max", Memory: "2048", Display: "gtk", VGA: "virtio", DebugOptions: "-s -S"},
		"Live CD Boot":        {Machine: "q35", Accel: "kvm", CPU: "host", Memory: "4096", Display: "gtk", VGA: "virtio", Boot: "d"},
		"Spice Desktop":       {Machine: "q35", Accel: "kvm", CPU: "host", Memory: "4096", Display: "spice-app", VGA: "qxl"},
		"Secure Boot EFI":     {Machine: "q35", Accel: "kvm", CPU: "host", Memory: "4096", Display: "gtk", VGA: "virtio", ExtraOptions: "-bios /usr/share/ovmf/OVMF.fd"},
	}

	var presetKeys []string
	for k := range presetsOpts {
		presetKeys = append(presetKeys, k)
	}

	presetSelect := widget.NewSelect(presetKeys, func(s string) {
		if p, ok := presetsOpts[s]; ok {
			*opts = *p
			refreshUI()
		}
	})
	presetSelect.PlaceHolder = "Select Preset..."

	runBtn := widget.NewButton("Run QEMU", func() {
		if err := opts.Validate(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		errCh := qemu.Run(context.Background(), opts)
		go func() {
			if err := <-errCh; err != nil {
				dialog.ShowError(err, w)
			}
		}()
	})

	bottomBar := container.NewHBox(
		presetSelect,
		layout.NewSpacer(),
		widget.NewLabel("Gophervisor QEMU Frontend"),
		layout.NewSpacer(),
		runBtn,
	)

	content := container.NewBorder(nil, bottomBar, nil, nil, tabs)
	w.SetContent(content)
	w.ShowAndRun()
}
