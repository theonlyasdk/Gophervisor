package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"Gophervisor/config"
)

func newFileEntry(w fyne.Window, entry *widget.Entry) fyne.CanvasObject {
	btn := widget.NewButton("Choose", func() {
		dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
			if uc != nil && err == nil {
				entry.SetText(uc.URI().Path())
			}
		}, w)
	})
	return container.NewBorder(nil, nil, nil, btn, entry)
}

func buildStandardTab(w fyne.Window, opts *config.Options) *container.TabItem {
	machineOptions := []string{"q35", "pc", "microvm", "isapc", "none"}
	machineSelect := widget.NewSelect(machineOptions, func(s string) { opts.Machine = s })
	machineSelect.SetSelected(opts.Machine)

	accelOptions := []string{"kvm", "hvf", "tcg", "xen", "whpx", "nvmm", ""}
	accelSelect := widget.NewSelect(accelOptions, func(s string) { opts.Accel = s })
	accelSelect.SetSelected(opts.Accel)

	cpuOptions := []string{"host", "max", "qemu64", "qemu32", "kvm64", "kvm32", "core2duo"}
	cpuSelect := widget.NewSelect(cpuOptions, func(s string) { opts.CPU = s })
	cpuSelect.SetSelected(opts.CPU)

	memEntry := widget.NewEntry()
	memEntry.PlaceHolder = "e.g., 2048, 4096, 8G"
	memEntry.SetText(opts.Memory)
	memEntry.OnChanged = func(s string) { opts.Memory = s }

	smpEntry := widget.NewEntry()
	smpEntry.PlaceHolder = "e.g., 4, cores=2,threads=2"
	smpEntry.SetText(opts.SMP)
	smpEntry.OnChanged = func(s string) { opts.SMP = s }

	bootEntry := widget.NewEntry()
	bootEntry.PlaceHolder = "e.g., c (HDD), d (CD-ROM)"
	bootEntry.SetText(opts.Boot)
	bootEntry.OnChanged = func(s string) { opts.Boot = s }

	form := widget.NewForm(
		&widget.FormItem{Text: "Machine (-machine)", Widget: machineSelect, HintText: "Select the emulated machine type"},
		&widget.FormItem{Text: "Accelerator (-accel)", Widget: accelSelect, HintText: "Hardware acceleration (KVM, HVF, etc.)"},
		&widget.FormItem{Text: "CPU (-cpu)", Widget: cpuSelect, HintText: "Select the emulated CPU model"},
		&widget.FormItem{Text: "Memory (-m)", Widget: memEntry, HintText: "Configuration for guest RAM"},
		&widget.FormItem{Text: "SMP (-smp)", Widget: smpEntry, HintText: "Set the number of initial CPUs"},
		&widget.FormItem{Text: "Boot Order (-boot)", Widget: bootEntry, HintText: "Specify boot drive priority"},
	)

	return container.NewTabItem("Standard", container.NewVScroll(form))
}

func newHDAFileEntry(w fyne.Window, entry *widget.Entry) fyne.CanvasObject {
	chooseBtn := widget.NewButton("Choose", func() {
		dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
			if uc != nil && err == nil {
				entry.SetText(uc.URI().Path())
			}
		}, w)
	})

	createBtn := widget.NewButton("Create", func() {
		showCreateDiskDialog(w, func(path string) {
			entry.SetText(path)
		})
	})

	buttons := container.NewHBox(chooseBtn, createBtn)
	return container.NewBorder(nil, nil, nil, buttons, entry)
}

func buildBlockTab(w fyne.Window, opts *config.Options) *container.TabItem {
	hdaEntry := widget.NewEntry()
	hdaEntry.PlaceHolder = "/path/to/image.img"
	hdaEntry.SetText(opts.DriveHDA)
	hdaEntry.OnChanged = func(s string) { opts.DriveHDA = s }

	cdromEntry := widget.NewEntry()
	cdromEntry.PlaceHolder = "/path/to/iso.iso"
	cdromEntry.SetText(opts.DriveCDROM)
	cdromEntry.OnChanged = func(s string) { opts.DriveCDROM = s }

	form := widget.NewForm(
		&widget.FormItem{Text: "HDA Image (-hda)", Widget: newHDAFileEntry(w, hdaEntry), HintText: "Hard disk 0 image file"},
		&widget.FormItem{Text: "CD-ROM Image (-cdrom)", Widget: newFileEntry(w, cdromEntry), HintText: "CD-ROM image file"},
	)
	return container.NewTabItem("Block Device", container.NewVScroll(form))
}

func buildDisplayTab(w fyne.Window, opts *config.Options) *container.TabItem {
	displayOptions := []string{"gtk", "sdl", "spice-app", "vnc", "none", "curses", "dbus", "egl-headless"}
	displaySelect := widget.NewSelect(displayOptions, func(s string) { opts.Display = s })
	displaySelect.SetSelected(opts.Display)

	vgaOptions := []string{"std", "cirrus", "vmware", "qxl", "xenfb", "tcx", "cg3", "virtio", "none"}
	vgaSelect := widget.NewSelect(vgaOptions, func(s string) { opts.VGA = s })
	vgaSelect.SetSelected(opts.VGA)

	form := widget.NewForm(
		&widget.FormItem{Text: "Display Backend (-display)", Widget: displaySelect, HintText: "Select display backend type"},
		&widget.FormItem{Text: "VGA Type (-vga)", Widget: vgaSelect, HintText: "Select video card type"},
	)
	return container.NewTabItem("Display", container.NewVScroll(form))
}

func buildNetworkTab(w fyne.Window, opts *config.Options) *container.TabItem {
	netdevOptions := []string{"user", "tap", "bridge", "vhost-user", "socket", "passt", "l2tpv3", "none"}
	netdevSelect := widget.NewSelect(netdevOptions, func(s string) { opts.Netdev = s })
	netdevSelect.SetSelected(opts.Netdev)

	form := widget.NewForm(
		&widget.FormItem{Text: "Netdev Backend (-netdev)", Widget: netdevSelect, HintText: "Configure a network backend"},
	)
	return container.NewTabItem("Network", container.NewVScroll(form))
}

func buildKernelTab(w fyne.Window, opts *config.Options) *container.TabItem {
	kernelEntry := widget.NewEntry()
	kernelEntry.PlaceHolder = "/path/to/bzImage"
	kernelEntry.SetText(opts.Kernel)
	kernelEntry.OnChanged = func(s string) { opts.Kernel = s }

	initrdEntry := widget.NewEntry()
	initrdEntry.PlaceHolder = "/path/to/initrd.img"
	initrdEntry.SetText(opts.Initrd)
	initrdEntry.OnChanged = func(s string) { opts.Initrd = s }

	appendEntry := widget.NewEntry()
	appendEntry.PlaceHolder = "root=/dev/sda1 console=ttyS0"
	appendEntry.SetText(opts.Append)
	appendEntry.OnChanged = func(s string) { opts.Append = s }

	form := widget.NewForm(
		&widget.FormItem{Text: "Kernel Image (-kernel)", Widget: newFileEntry(w, kernelEntry), HintText: "Use bzImage as kernel image"},
		&widget.FormItem{Text: "Initrd (-initrd)", Widget: newFileEntry(w, initrdEntry), HintText: "Use file as initial ram disk"},
		&widget.FormItem{Text: "Append line (-append)", Widget: appendEntry, HintText: "Kernel command line arguments"},
	)
	return container.NewTabItem("Kernel", container.NewVScroll(form))
}

func buildMiscTab(w fyne.Window, opts *config.Options) *container.TabItem {
	usbCheck := widget.NewCheck("Enable USB (-usb)", func(b bool) {
		opts.USB = b
	})
	usbCheck.Checked = opts.USB

	usbDevEntry := widget.NewEntry()
	usbDevEntry.PlaceHolder = "e.g., tablet, host:1234:5678"
	usbDevEntry.SetText(opts.USBDevice)
	usbDevEntry.OnChanged = func(s string) { opts.USBDevice = s }

	tpmEntry := widget.NewEntry()
	tpmEntry.PlaceHolder = "passthrough,id=tpm0,path=/dev/tpm0"
	tpmEntry.SetText(opts.TPMDev)
	tpmEntry.OnChanged = func(s string) { opts.TPMDev = s }

	charEntry := widget.NewEntry()
	charEntry.PlaceHolder = "stdio,id=char0"
	charEntry.SetText(opts.CharDev)
	charEntry.OnChanged = func(s string) { opts.CharDev = s }

	form := widget.NewForm(
		&widget.FormItem{Text: "USB Device (-usbdevice)", Widget: usbDevEntry, HintText: "Add USB device by name/ID"},
		&widget.FormItem{Text: "TPM Device (-tpmdev)", Widget: tpmEntry, HintText: "Configure TPM device"},
		&widget.FormItem{Text: "Char Device (-chardev)", Widget: charEntry, HintText: "Configure character device backend"},
	)

	content := container.NewVBox(
		usbCheck,
		form,
	)

	return container.NewTabItem("Misc", container.NewVScroll(content))
}

func buildAdvancedTab(w fyne.Window, opts *config.Options) *container.TabItem {
	debugEntry := widget.NewEntry()
	debugEntry.PlaceHolder = "-s -S"
	debugEntry.SetText(opts.DebugOptions)
	debugEntry.OnChanged = func(s string) { opts.DebugOptions = s }

	objEntry := widget.NewEntry()
	objEntry.PlaceHolder = "rng-random,id=rng0,filename=/dev/urandom"
	objEntry.SetText(opts.GenericObj)
	objEntry.OnChanged = func(s string) { opts.GenericObj = s }

	extraEntry := widget.NewMultiLineEntry()
	extraEntry.PlaceHolder = "-no-reboot -daemonize"
	extraEntry.SetText(opts.ExtraOptions)
	extraEntry.OnChanged = func(s string) { opts.ExtraOptions = s }

	form := widget.NewForm(
		&widget.FormItem{Text: "Debug Options (-d, -s, -S)", Widget: debugEntry, HintText: "Low-level debug flags"},
		&widget.FormItem{Text: "Generic Object (-object)", Widget: objEntry, HintText: "Create a generic QEMU object"},
		&widget.FormItem{Text: "Extra Custom Args", Widget: extraEntry, HintText: "Any other QEMU arguments"},
	)

	return container.NewTabItem("Advanced", container.NewVScroll(form))
}
