package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"Gophervisor/config"
)

func withTopSpacing(obj fyne.CanvasObject) fyne.CanvasObject {
	topGap := canvas.NewRectangle(color.Transparent)
	topGap.SetMinSize(fyne.NewSize(0, 10))
	return container.NewBorder(topGap, nil, nil, nil, obj)
}

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

	return container.NewTabItem("General", withTopSpacing(container.NewVScroll(form)))
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
	return container.NewTabItem("Block Device", withTopSpacing(container.NewVScroll(form)))
}

func buildDisplayTab(w fyne.Window, opts *config.Options, onVNCChanged func(bool)) *container.TabItem {
	displayOptions := []string{"gtk", "sdl", "spice-app", "vnc", "none", "curses", "dbus", "egl-headless"}
	displaySelect := widget.NewSelect(displayOptions, nil)

	vgaOptions := []string{"std", "cirrus", "vmware", "qxl", "xenfb", "tcx", "cg3", "virtio", "none"}
	vgaSelect := widget.NewSelect(vgaOptions, func(s string) { opts.VGA = s })
	vgaSelect.SetSelected(opts.VGA)

	vncAddrEntry := widget.NewEntry()
	vncAddrEntry.PlaceHolder = "localhost"
	vncAddrEntry.SetText(opts.VNCAddress)
	if strings.TrimSpace(vncAddrEntry.Text) == "" {
		vncAddrEntry.SetText("localhost")
	}

	vncDisplayEntry := widget.NewEntry()
	vncDisplayEntry.PlaceHolder = "1"
	vncDisplayEntry.SetText(opts.VNCDisplay)
	if strings.TrimSpace(vncDisplayEntry.Text) == "" {
		vncDisplayEntry.SetText("1")
	}

	vncPasswordCheck := widget.NewCheck("Require VNC password (password=on)", func(b bool) {
		opts.VNCPassword = b
		if strings.TrimSpace(displaySelect.Selected) == "vnc" {
			opts.Display = buildVNCDisplaySpec(opts)
		}
	})
	vncPasswordCheck.Checked = opts.VNCPassword

	vncWebPortEntry := widget.NewEntry()
	vncWebPortEntry.PlaceHolder = "6080"
	vncWebPortEntry.SetText(opts.VNCWebPort)
	if strings.TrimSpace(vncWebPortEntry.Text) == "" {
		vncWebPortEntry.SetText("6080")
	}

	vncForm := widget.NewForm(
		&widget.FormItem{Text: "VNC Address", Widget: vncAddrEntry, HintText: "Listen address for QEMU VNC (host in vnc=host:N)"},
		&widget.FormItem{Text: "VNC Display #", Widget: vncDisplayEntry, HintText: "Display number N (TCP port 5900+N)"},
		&widget.FormItem{Text: "noVNC Web Port", Widget: vncWebPortEntry, HintText: "Web socket proxy port for browser access"},
		&widget.FormItem{Text: "Security", Widget: vncPasswordCheck, HintText: "Enable VNC password requirement"},
	)

	updateDisplay := func() {
		isVNC := strings.TrimSpace(displaySelect.Selected) == "vnc"
		if isVNC {
			opts.VNCAddress = strings.TrimSpace(vncAddrEntry.Text)
			opts.VNCDisplay = strings.TrimSpace(vncDisplayEntry.Text)
			opts.VNCWebPort = strings.TrimSpace(vncWebPortEntry.Text)
			opts.Display = buildVNCDisplaySpec(opts)
			vncForm.Show()
		} else {
			opts.Display = displaySelect.Selected
			vncForm.Hide()
		}
		if onVNCChanged != nil {
			onVNCChanged(isVNC)
		}
	}

	displaySelect.OnChanged = func(string) { updateDisplay() }
	vncAddrEntry.OnChanged = func(s string) {
		opts.VNCAddress = s
		updateDisplay()
	}
	vncDisplayEntry.OnChanged = func(s string) {
		opts.VNCDisplay = s
		updateDisplay()
	}
	vncWebPortEntry.OnChanged = func(s string) {
		opts.VNCWebPort = s
	}

	initialDisplay := strings.TrimSpace(opts.Display)
	if strings.HasPrefix(initialDisplay, "vnc") {
		displaySelect.SetSelected("vnc")
	} else {
		displaySelect.SetSelected(initialDisplay)
		if displaySelect.Selected == "" {
			displaySelect.SetSelected("gtk")
		}
	}
	updateDisplay()

	form := widget.NewForm(
		&widget.FormItem{Text: "Display Backend (-display)", Widget: displaySelect, HintText: "Select display backend type"},
		&widget.FormItem{Text: "VGA Type (-vga)", Widget: vgaSelect, HintText: "Select video card type"},
	)
	content := container.NewVBox(form, vncForm)
	return container.NewTabItem("Display", withTopSpacing(container.NewVScroll(content)))
}

func buildNetworkTab(w fyne.Window, opts *config.Options) *container.TabItem {
	netdevOptions := []string{"user", "tap", "bridge", "vhost-user", "socket", "passt", "l2tpv3", "none"}
	netdevSelect := widget.NewSelect(netdevOptions, nil)

	idEntry := widget.NewEntry()
	idEntry.PlaceHolder = "id (e.g., net0)"
	if strings.Contains(opts.Netdev, "id=") {
		for _, part := range strings.Split(opts.Netdev, ",") {
			if strings.HasPrefix(part, "id=") {
				idEntry.SetText(strings.TrimPrefix(part, "id="))
				break
			}
		}
	}

	param1Entry := widget.NewEntry()
	param2Entry := widget.NewEntry()
	param1 := &widget.FormItem{Text: "", Widget: param1Entry}
	param2 := &widget.FormItem{Text: "", Widget: param2Entry}

	updateNetdevValue := func() {
		backend := strings.TrimSpace(netdevSelect.Selected)
		if backend == "" || backend == "none" {
			opts.Netdev = backend
			return
		}

		args := []string{backend}
		if id := strings.TrimSpace(idEntry.Text); id != "" {
			args = append(args, "id="+id)
		}
		if p1 := strings.TrimSpace(param1Entry.Text); p1 != "" {
			args = append(args, p1)
		}
		if p2 := strings.TrimSpace(param2Entry.Text); p2 != "" {
			args = append(args, p2)
		}
		opts.Netdev = strings.Join(args, ",")
	}

	updateBackendFields := func(backend string) {
		param1Entry.SetText("")
		param2Entry.SetText("")
		param1Entry.PlaceHolder = ""
		param2Entry.PlaceHolder = ""

		switch backend {
		case "user":
			param1.Text = "Host Forward"
			param1.HintText = "e.g., hostfwd=tcp::2222-:22"
			param1Entry.PlaceHolder = "hostfwd=tcp::2222-:22"
			param2.Text = "DNS"
			param2.HintText = "Optional custom DNS server"
			param2Entry.PlaceHolder = "dns=1.1.1.1"
			param1.Widget.Show()
			param2.Widget.Show()
		case "tap":
			param1.Text = "Interface"
			param1.HintText = "Tap interface name"
			param1Entry.PlaceHolder = "ifname=tap0"
			param2.Text = "Script"
			param2.HintText = "Tap setup script"
			param2Entry.PlaceHolder = "script=/etc/qemu-ifup"
			param1.Widget.Show()
			param2.Widget.Show()
		case "bridge":
			param1.Text = "Bridge"
			param1.HintText = "Bridge device name"
			param1Entry.PlaceHolder = "br=br0"
			param2.Text = "Helper"
			param2.HintText = "Bridge helper binary"
			param2Entry.PlaceHolder = "helper=/usr/lib/qemu/qemu-bridge-helper"
			param1.Widget.Show()
			param2.Widget.Show()
		case "socket":
			param1.Text = "Listen/Connect"
			param1.HintText = "Socket endpoint mode"
			param1Entry.PlaceHolder = "listen=:1234 (or connect=127.0.0.1:1234)"
			param2.Text = "Mcast"
			param2.HintText = "Optional multicast endpoint"
			param2Entry.PlaceHolder = "mcast=230.0.0.1:1234"
			param1.Widget.Show()
			param2.Widget.Show()
		case "vhost-user":
			param1.Text = "Char Device ID"
			param1.HintText = "Attach to pre-defined chardev"
			param1Entry.PlaceHolder = "chardev=char0"
			param2.Text = "Vhostforce"
			param2.HintText = "Optional flag"
			param2Entry.PlaceHolder = "vhostforce=on"
			param1.Widget.Show()
			param2.Widget.Show()
		case "passt":
			param1.Text = "Path"
			param1.HintText = "Path to passt socket or executable config"
			param1Entry.PlaceHolder = "path=/run/user/1000/passt.sock"
			param2.Text = "MTU"
			param2.HintText = "Optional MTU setting"
			param2Entry.PlaceHolder = "mtu=1500"
			param1.Widget.Show()
			param2.Widget.Show()
		case "l2tpv3":
			param1.Text = "Src"
			param1.HintText = "Source endpoint"
			param1Entry.PlaceHolder = "src=192.168.1.10"
			param2.Text = "Dst"
			param2.HintText = "Destination endpoint"
			param2Entry.PlaceHolder = "dst=192.168.1.11"
			param1.Widget.Show()
			param2.Widget.Show()
		default:
			param1.Widget.Hide()
			param2.Widget.Hide()
		}

		updateNetdevValue()
	}

	netdevSelect.OnChanged = func(s string) { updateBackendFields(s) }
	idEntry.OnChanged = func(string) { updateNetdevValue() }
	param1Entry.OnChanged = func(string) { updateNetdevValue() }
	param2Entry.OnChanged = func(string) { updateNetdevValue() }

	initialBackend := strings.TrimSpace(strings.Split(opts.Netdev, ",")[0])
	if initialBackend == "" {
		initialBackend = "user"
	}
	validBackend := false
	for _, v := range netdevOptions {
		if v == initialBackend {
			validBackend = true
			break
		}
	}
	if !validBackend {
		initialBackend = "user"
	}
	netdevSelect.SetSelected(initialBackend)
	updateBackendFields(initialBackend)

	form := widget.NewForm(
		&widget.FormItem{Text: "Netdev Backend (-netdev)", Widget: netdevSelect, HintText: "Configure a network backend"},
		&widget.FormItem{Text: "Netdev ID", Widget: idEntry, HintText: "Required when backend is not 'none'"},
		param1,
		param2,
	)

	// Rebind after wrapping to ensure preview refreshes on all changes.
	netdevSelect.OnChanged = func(s string) {
		updateBackendFields(s)
	}
	idEntry.OnChanged = func(string) { updateNetdevValue() }
	param1Entry.OnChanged = func(string) { updateNetdevValue() }
	param2Entry.OnChanged = func(string) { updateNetdevValue() }

	return container.NewTabItem("Network", withTopSpacing(container.NewVScroll(form)))
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
	return container.NewTabItem("Kernel", withTopSpacing(container.NewVScroll(form)))
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

	return container.NewTabItem("Misc", withTopSpacing(container.NewVScroll(content)))
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

	return container.NewTabItem("Advanced", withTopSpacing(container.NewVScroll(form)))
}
