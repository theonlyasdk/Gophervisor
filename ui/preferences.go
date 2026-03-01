package ui

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	prefQemuSystemPath = "qemu_system_path"
	prefQemuImgPath    = "qemu_img_path"
)

func qemuSystemBinaryPath() string {
	return configuredOrDetectedBinaryPath(prefQemuSystemPath, "qemu-system-x86_64", "qemu-system-x86_64.exe")
}

func qemuImgBinaryPath() string {
	return configuredOrDetectedBinaryPath(prefQemuImgPath, "qemu-img", "qemu-img.exe")
}

func configuredOrDetectedBinaryPath(prefKey string, names ...string) string {
	p := fyne.CurrentApp().Preferences()
	saved := strings.TrimSpace(p.StringWithFallback(prefKey, ""))
	if binaryExists(saved) {
		return saved
	}
	if detected := detectBinaryPath(names...); detected != "" {
		return detected
	}
	return saved
}

func detectBinaryPath(names ...string) string {
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}

	candidates := []string{}
	switch runtime.GOOS {
	case "windows":
		candidates = append(candidates,
			`C:\Program Files\qemu\qemu-system-x86_64.exe`,
			`C:\Program Files\qemu\qemu-img.exe`,
			`C:\Program Files (x86)\qemu\qemu-system-x86_64.exe`,
			`C:\Program Files (x86)\qemu\qemu-img.exe`,
		)
	case "darwin":
		candidates = append(candidates,
			"/opt/homebrew/bin/qemu-system-x86_64",
			"/opt/homebrew/bin/qemu-img",
			"/usr/local/bin/qemu-system-x86_64",
			"/usr/local/bin/qemu-img",
		)
	default:
		candidates = append(candidates,
			"/usr/bin/qemu-system-x86_64",
			"/usr/bin/qemu-img",
			"/usr/local/bin/qemu-system-x86_64",
			"/usr/local/bin/qemu-img",
		)
	}

	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	for _, c := range candidates {
		if !nameSet[filepath.Base(c)] {
			continue
		}
		if binaryExists(c) {
			return c
		}
	}
	return ""
}

func binaryExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func showQemuDownloadHelperDialog(parent fyne.Window) {
	content := container.NewVBox(
		widget.NewLabel("QEMU is required to run virtual machines and create disk images."),
		widget.NewSeparator(),
		widget.NewLabel("Linux (Debian/Ubuntu): sudo apt install qemu-system-x86 qemu-utils"),
		widget.NewLabel("Linux (Arch): sudo pacman -S qemu-desktop"),
		widget.NewLabel("Linux (Fedora): sudo dnf install @virtualization qemu-img"),
		widget.NewLabel("Linux (RHEL/CentOS Stream): sudo dnf install qemu-kvm qemu-img"),
		widget.NewLabel("Linux (openSUSE): sudo zypper install qemu-x86 qemu-tools"),
		widget.NewLabel("Linux (Alpine): sudo apk add qemu-system-x86_64 qemu-img"),
		widget.NewLabel("Linux (NixOS/Nix): nix-env -iA nixpkgs.qemu"),
		widget.NewLabel("macOS (Homebrew): brew install qemu"),
		widget.NewLabel("Windows (winget): winget install SoftwareFreedomConservancy.QEMU"),
	)

	openDownloads := widget.NewButton("Open QEMU Downloads", func() {
		u, _ := url.Parse("https://www.qemu.org/download/")
		if err := fyne.CurrentApp().OpenURL(u); err != nil {
			dialog.ShowError(fmt.Errorf("failed to open browser: %w", err), parent)
		}
	})

	d := dialog.NewCustom("QEMU Download Helper", "Close", container.NewBorder(nil, openDownloads, nil, nil, content), parent)
	d.Resize(fyne.NewSize(760, 360))
	d.Show()
}

func showPreferencesDialog(parent fyne.Window) {
	prefs := fyne.CurrentApp().Preferences()

	systemPath := strings.TrimSpace(prefs.StringWithFallback(prefQemuSystemPath, ""))
	if systemPath == "" {
		systemPath = detectBinaryPath("qemu-system-x86_64", "qemu-system-x86_64.exe")
	}

	imgPath := strings.TrimSpace(prefs.StringWithFallback(prefQemuImgPath, ""))
	if imgPath == "" {
		imgPath = detectBinaryPath("qemu-img", "qemu-img.exe")
	}

	systemEntry := widget.NewEntry()
	systemEntry.SetText(systemPath)
	systemEntry.PlaceHolder = "Path to qemu-system-x86_64"

	imgEntry := widget.NewEntry()
	imgEntry.SetText(imgPath)
	imgEntry.PlaceHolder = "Path to qemu-img"

	systemDownloadBtn := widget.NewButton("Download", func() {
		showQemuDownloadHelperDialog(parent)
	})
	imgDownloadBtn := widget.NewButton("Download", func() {
		showQemuDownloadHelperDialog(parent)
	})

	updateDownloadButtonState := func() {
		if binaryExists(systemEntry.Text) {
			systemDownloadBtn.Hide()
		} else {
			systemDownloadBtn.Show()
		}
		if binaryExists(imgEntry.Text) {
			imgDownloadBtn.Hide()
		} else {
			imgDownloadBtn.Show()
		}
	}

	systemEntry.OnChanged = func(string) { updateDownloadButtonState() }
	imgEntry.OnChanged = func(string) { updateDownloadButtonState() }
	updateDownloadButtonState()

	systemRow := container.NewBorder(nil, nil, nil, systemDownloadBtn, systemEntry)
	imgRow := container.NewBorder(nil, nil, nil, imgDownloadBtn, imgEntry)

	form := widget.NewForm(
		&widget.FormItem{Text: "QEMU Location", Widget: systemRow},
		&widget.FormItem{Text: "QEMU Img Location", Widget: imgRow},
	)

	d := dialog.NewCustomConfirm("Preferences", "Save", "Cancel", container.NewPadded(form), func(save bool) {
		if !save {
			return
		}
		prefs.SetString(prefQemuSystemPath, strings.TrimSpace(systemEntry.Text))
		prefs.SetString(prefQemuImgPath, strings.TrimSpace(imgEntry.Text))
	}, parent)
	d.Resize(fyne.NewSize(700, 220))
	d.Show()
}
