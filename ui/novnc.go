package ui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image/color"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/net/websocket"

	"Gophervisor/config"
	"Gophervisor/qemu"
)

var errGitNotFound = errors.New("git not found")

func isVNCDisplaySelected(opts *config.Options) bool {
	return strings.HasPrefix(strings.TrimSpace(opts.Display), "vnc")
}

func buildVNCDisplaySpec(opts *config.Options) string {
	host := strings.TrimSpace(opts.VNCAddress)
	if host == "" {
		host = "localhost"
	}
	display := strings.TrimSpace(opts.VNCDisplay)
	if display == "" {
		display = "1"
	}
	spec := fmt.Sprintf("vnc=%s:%s", host, display)
	if opts.VNCPassword {
		spec += ",password=on"
	}
	return spec
}

func vncTCPPort(opts *config.Options) (string, int, error) {
	host := strings.TrimSpace(opts.VNCAddress)
	if host == "" {
		host = "localhost"
	}
	display := strings.TrimSpace(opts.VNCDisplay)
	if display == "" {
		display = "1"
	}
	displayNum, err := strconv.Atoi(display)
	if err != nil || displayNum < 0 {
		return "", 0, fmt.Errorf("invalid VNC display number: %q", display)
	}
	return host, 5900 + displayNum, nil
}

func gophervisorDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "Gophervisor"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "gophervisor"), nil
	default:
		return filepath.Join(home, ".gophervisor"), nil
	}
}

func ensureNoVNCAssetsAvailable(ctx context.Context, update func(string)) (string, error) {
	if update == nil {
		update = func(string) {}
	}

	dataDir, err := gophervisorDataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}
	noVNCDir := filepath.Join(dataDir, "novnc")
	vncHTML := filepath.Join(noVNCDir, "vnc.html")

	if _, err := os.Stat(vncHTML); err == nil {
		update("Using cached noVNC assets.")
		return noVNCDir, nil
	}
	if _, err := exec.LookPath("git"); err != nil {
		return "", errGitNotFound
	}

	if _, err := os.Stat(noVNCDir); err == nil {
		backupDir := noVNCDir + "-broken-" + strconv.FormatInt(time.Now().Unix(), 10)
		update("Existing incomplete noVNC directory detected. Moving aside for a clean clone.")
		if err := os.Rename(noVNCDir, backupDir); err != nil {
			return "", fmt.Errorf("failed to move broken noVNC directory: %w", err)
		}
	}

	update("Cloning noVNC repository...")
	clone := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "https://github.com/novnc/noVNC.git", noVNCDir)
	if err := runCommandWithProgress(ctx, clone, update); err != nil {
		return "", fmt.Errorf("failed to clone noVNC: %w", err)
	}

	if _, err := os.Stat(vncHTML); err != nil {
		return "", fmt.Errorf("noVNC assets not found at %s", vncHTML)
	}
	return noVNCDir, nil
}

func runCommandWithProgress(ctx context.Context, cmd *exec.Cmd, update func(string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	lines := make(chan string, 128)
	var wg sync.WaitGroup
	readStream := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		if scanErr := scanner.Err(); scanErr != nil {
			lines <- fmt.Sprintf("stream error: %v", scanErr)
		}
	}

	wg.Add(2)
	go readStream(stdout)
	go readStream(stderr)
	go func() {
		wg.Wait()
		close(lines)
	}()

	var tail []string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line, ok := <-lines:
			if !ok {
				waitErr := cmd.Wait()
				if waitErr != nil {
					if len(tail) > 0 {
						return fmt.Errorf("%w\n%s", waitErr, strings.Join(tail, "\n"))
					}
					return waitErr
				}
				return nil
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			update(line)
			tail = append(tail, line)
			if len(tail) > 30 {
				tail = tail[1:]
			}
		}
	}
}

func showGitInstallHelperDialog(parent fyne.Window) {
	content := container.NewVBox(
		widget.NewLabel("Git is required to download noVNC assets automatically."),
		widget.NewSeparator(),
		widget.NewLabel("Linux (Debian/Ubuntu): sudo apt install git"),
		widget.NewLabel("Linux (Arch): sudo pacman -S git"),
		widget.NewLabel("Linux (Fedora/RHEL): sudo dnf install git"),
		widget.NewLabel("macOS (Homebrew): brew install git"),
		widget.NewLabel("Windows (winget): winget install --id Git.Git -e"),
	)
	d := dialog.NewCustom("Git Not Found", "Close", container.NewPadded(content), parent)
	d.Resize(fyne.NewSize(640, 240))
	d.Show()
}

func showNoVNCStartupError(parent fyne.Window, err error, details string) {
	details = strings.TrimSpace(details)
	if details == "" {
		dialog.ShowError(err, parent)
		return
	}

	detailBox := widget.NewMultiLineEntry()
	detailBox.SetText(details)
	detailBox.SetMinRowsVisible(10)
	detailBox.Wrapping = fyne.TextWrapOff
	detailBox.TextStyle = fyne.TextStyle{Monospace: true}

	content := container.NewVBox(
		widget.NewLabel(err.Error()),
		widget.NewSeparator(),
		widget.NewLabel("QEMU output:"),
		detailBox,
	)
	d := dialog.NewCustom("noVNC Startup Error", "Close", container.NewPadded(content), parent)
	d.Resize(fyne.NewSize(840, 420))
	d.Show()
}

func compactOutput(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lines := strings.Split(raw, "\n")
	if len(lines) > 120 {
		lines = lines[len(lines)-120:]
	}
	return strings.Join(lines, "\n")
}

type noVNCProgressDialog struct {
	dialog    *dialog.CustomDialog
	status    *widget.Label
	progress  *widget.ProgressBar
	log       *widget.Entry
	stopBtn   *widget.Button
	qrBtn     *widget.Button
	publicURL string

	mu        sync.Mutex
	lines     []string
	closeOnce sync.Once
}

type noVNCSession struct {
	qemuCmd *exec.Cmd
	server  *noVNCServer
}

type noVNCLogTheme struct {
	fyne.Theme
}

func (t *noVNCLogTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameForeground {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if name == theme.ColorNameDisabled {
		return color.NRGBA{R: 220, G: 220, B: 220, A: 255}
	}
	return t.Theme.Color(name, variant)
}

var (
	noVNCSessionMu     sync.Mutex
	activeNoVNCSession *noVNCSession
)

func newNoVNCProgressDialog(parent fyne.Window, onStop func()) *noVNCProgressDialog {
	status := widget.NewLabel("Preparing noVNC...")
	progress := widget.NewProgressBar()
	progress.Min = 0
	progress.Max = 1
	progress.SetValue(0)
	stopBtn := widget.NewButton("Stop noVNC", func() {
		if onStop != nil {
			onStop()
		}
	})
	stopBtn.Disable()
	qrBtn := widget.NewButton("Show Public URL QR Code", nil)
	qrBtn.Disable()
	logView := widget.NewMultiLineEntry()
	logView.SetMinRowsVisible(12)
	logView.Wrapping = fyne.TextWrapOff
	logView.TextStyle = fyne.TextStyle{Monospace: true}
	logWrapped := container.NewThemeOverride(logView, &noVNCLogTheme{Theme: theme.DefaultTheme()})

	p := &noVNCProgressDialog{
		status:   status,
		progress: progress,
		log:      logView,
		stopBtn:  stopBtn,
		qrBtn:    qrBtn,
	}
	qrBtn.OnTapped = func() {
		p.mu.Lock()
		publicURL := p.publicURL
		p.mu.Unlock()
		if strings.TrimSpace(publicURL) == "" {
			return
		}
		showPublicURLQRCodeDialog(parent, publicURL)
	}

	content := container.NewBorder(
		container.NewVBox(container.NewBorder(nil, nil, nil, container.NewHBox(qrBtn, stopBtn), status), progress, widget.NewSeparator()),
		nil,
		nil,
		nil,
		logWrapped,
	)

	d := dialog.NewCustom("Starting noVNC", "Close", container.NewPadded(content), parent)
	d.Resize(fyne.NewSize(760, 360))
	p.dialog = d
	return p
}

func (p *noVNCProgressDialog) Show() {
	p.dialog.Show()
}

func (p *noVNCProgressDialog) Close() {
	p.closeOnce.Do(func() {
		fyne.Do(func() {
			p.dialog.Hide()
		})
	})
}

func (p *noVNCProgressDialog) SetStatus(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	fyne.Do(func() {
		p.status.SetText(message)
	})
}

func (p *noVNCProgressDialog) AppendLog(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	p.mu.Lock()
	p.lines = append(p.lines, line)
	if len(p.lines) > 120 {
		p.lines = p.lines[len(p.lines)-120:]
	}
	text := strings.Join(p.lines, "\n")
	p.mu.Unlock()

	fyne.Do(func() {
		p.log.SetText(text)
		if len(p.lines) > 0 {
			p.log.CursorRow = len(p.lines) - 1
		}
		p.log.CursorColumn = 0
		p.log.Refresh()
	})
}

func (p *noVNCProgressDialog) SetProgress(currentStep int, totalSteps int) {
	if totalSteps < 1 {
		return
	}
	if currentStep < 0 {
		currentStep = 0
	}
	if currentStep > totalSteps {
		currentStep = totalSteps
	}
	value := float64(currentStep) / float64(totalSteps)
	fyne.Do(func() {
		p.progress.SetValue(value)
	})
}

func (p *noVNCProgressDialog) SetStopEnabled(enabled bool) {
	fyne.Do(func() {
		if enabled {
			p.stopBtn.Enable()
			return
		}
		p.stopBtn.Disable()
	})
}

func (p *noVNCProgressDialog) SetPublicURL(publicURL string) {
	publicURL = strings.TrimSpace(publicURL)
	p.mu.Lock()
	p.publicURL = publicURL
	p.mu.Unlock()

	fyne.Do(func() {
		if publicURL == "" {
			p.qrBtn.Disable()
			return
		}
		p.qrBtn.Enable()
	})
}

func noVNCWebListenHost(opts *config.Options) string {
	_ = opts
	return "0.0.0.0"
}

func noVNCBrowserHost(listenHost string) string {
	switch strings.TrimSpace(listenHost) {
	case "", "0.0.0.0", "::", "[::]":
		return "localhost"
	default:
		return listenHost
	}
}

func vncBridgeDialHost(vncHost string) string {
	switch strings.TrimSpace(vncHost) {
	case "", "0.0.0.0", "::", "[::]":
		return "localhost"
	default:
		return strings.TrimSpace(vncHost)
	}
}

func normalizePort(raw string, fallback string) (string, error) {
	port := strings.TrimSpace(raw)
	if port == "" {
		port = fallback
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid port: %q", port)
	}
	return strconv.Itoa(n), nil
}

type noVNCServer struct {
	httpServer *http.Server
	listenHost string
	listenPort string
}

func startNoVNCServer(noVNCDir string, listenHost string, webPort string, targetHost string, targetPort int, logf func(string)) (*noVNCServer, error) {
	if logf == nil {
		logf = func(string) {}
	}

	targetAddr := net.JoinHostPort(targetHost, strconv.Itoa(targetPort))
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir(noVNCDir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/vnc.html", http.StatusTemporaryRedirect)
			return
		}
		fileServer.ServeHTTP(w, r)
	}))

	wsHandler := websocket.Server{
		Handshake: func(*websocket.Config, *http.Request) error {
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			bridgeVNCOverWebSocket(ws, targetAddr, logf)
		}),
	}
	mux.Handle("/websockify", wsHandler)

	hostsToTry := []string{strings.TrimSpace(listenHost), "localhost", "127.0.0.1", "0.0.0.0"}
	tried := make(map[string]struct{})
	var (
		ln         net.Listener
		err        error
		actualHost string
		lastErr    error
	)
	for _, h := range hostsToTry {
		if h == "" {
			continue
		}
		if _, seen := tried[h]; seen {
			continue
		}
		tried[h] = struct{}{}
		listenAddr := net.JoinHostPort(h, webPort)
		ln, err = net.Listen("tcp", listenAddr)
		if err == nil {
			actualHost = h
			break
		}
		lastErr = err
		logf(fmt.Sprintf("Bind attempt failed for %s: %v", listenAddr, err))
	}
	if ln == nil {
		return nil, fmt.Errorf("failed to start noVNC web server on port %s: %w", webPort, lastErr)
	}

	srv := &http.Server{
		Handler: mux,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logf(fmt.Sprintf("noVNC web server stopped with error: %v", err))
		}
	}()

	logf(fmt.Sprintf("Serving noVNC assets from %s", noVNCDir))
	logf(fmt.Sprintf("noVNC HTTP server listening on %s:%s", actualHost, webPort))
	logf(fmt.Sprintf("Forwarding WebSocket traffic to VNC target %s", targetAddr))
	return &noVNCServer{httpServer: srv, listenHost: actualHost, listenPort: webPort}, nil
}

func (s *noVNCServer) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func bridgeVNCOverWebSocket(ws *websocket.Conn, targetAddr string, logf func(string)) {
	defer ws.Close()

	tcpConn, err := dialWithRetry(targetAddr, 4*time.Second)
	if err != nil {
		logf(fmt.Sprintf("failed to connect to VNC target %s: %v", targetAddr, err))
		return
	}
	defer tcpConn.Close()

	ws.PayloadType = websocket.BinaryFrame
	errCh := make(chan error, 2)
	go func() {
		_, copyErr := io.Copy(tcpConn, ws)
		errCh <- copyErr
	}()
	go func() {
		ws.PayloadType = websocket.BinaryFrame
		_, copyErr := io.Copy(ws, tcpConn)
		errCh <- copyErr
	}()

	firstErr := <-errCh
	_ = tcpConn.Close()
	_ = ws.Close()
	secondErr := <-errCh

	if err := firstSignificantError(firstErr, secondErr); err != nil {
		logf(fmt.Sprintf("VNC websocket bridge closed with error: %v", err))
	}
}

func dialWithRetry(address string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		conn, err := net.DialTimeout("tcp", address, 700*time.Millisecond)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return nil, lastErr
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func firstSignificantError(errs ...error) error {
	for _, err := range errs {
		if err == nil || errors.Is(err, io.EOF) {
			continue
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "use of closed network connection") ||
			strings.Contains(msg, "broken pipe") ||
			strings.Contains(msg, "connection reset by peer") {
			continue
		}
		return err
	}
	return nil
}

func buildNoVNCURL(listenHost string, webPort string) *url.URL {
	browserHost := noVNCBrowserHost(listenHost)
	query := url.Values{}
	query.Set("autoconnect", "true")
	query.Set("resize", "scale")
	query.Set("host", browserHost)
	query.Set("port", webPort)
	query.Set("path", "websockify")

	return &url.URL{
		Scheme:   "http",
		Host:     net.JoinHostPort(browserHost, webPort),
		Path:     "/vnc.html",
		RawQuery: query.Encode(),
	}
}

func localLANIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			private := ip.IsPrivate()
			if private {
				return ip.String()
			}
		}
	}
	return ""
}

func showPublicURLQRCodeDialog(parent fyne.Window, publicURL string) {
	escaped := url.QueryEscape(publicURL)
	qrSource := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=360x360&data=%s", escaped)
	qrURI, err := storage.ParseURI(qrSource)

	urlEntry := widget.NewEntry()
	urlEntry.SetText(publicURL)
	urlEntry.TextStyle = fyne.TextStyle{Monospace: true}

	var content fyne.CanvasObject
	if err == nil {
		img := canvas.NewImageFromURI(qrURI)
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(fyne.NewSize(340, 340))
		content = container.NewVBox(
			widget.NewLabel("Scan to open noVNC from another device on your local network."),
			img,
			widget.NewLabel("Public URL:"),
			urlEntry,
		)
	} else {
		content = container.NewVBox(
			widget.NewLabel("Unable to render QR image. Use this URL manually:"),
			urlEntry,
		)
	}

	d := dialog.NewCustom("Public URL QR Code", "Close", container.NewPadded(content), parent)
	d.Resize(fyne.NewSize(500, 520))
	d.Show()
}

func waitForHTTPServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 800 * time.Millisecond}
	var lastErr error

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("timed out waiting for noVNC HTTP server")
}

func waitForTCPServer(address string, timeout time.Duration, qemuExit <-chan error) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if qemuExit != nil {
			select {
			case err, ok := <-qemuExit:
				if ok {
					if err != nil {
						return fmt.Errorf("QEMU exited before VNC became ready: %w", err)
					}
					return errors.New("QEMU exited before VNC became ready")
				}
				return errors.New("QEMU exited before VNC became ready")
			default:
			}
		}

		conn, err := net.DialTimeout("tcp", address, 600*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}

	if lastErr != nil {
		return fmt.Errorf("timed out waiting for %s: %w", address, lastErr)
	}
	return fmt.Errorf("timed out waiting for %s", address)
}

func isNoVNCSessionRunning() bool {
	noVNCSessionMu.Lock()
	defer noVNCSessionMu.Unlock()
	return activeNoVNCSession != nil
}

func stopNoVNCSession() {
	noVNCSessionMu.Lock()
	session := activeNoVNCSession
	activeNoVNCSession = nil
	noVNCSessionMu.Unlock()

	if session == nil {
		return
	}

	if session.qemuCmd != nil && session.qemuCmd.Process != nil {
		_ = session.qemuCmd.Process.Kill()
	}
	if session.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = session.server.Shutdown(ctx)
	}
}

func setNoVNCSession(session *noVNCSession) {
	noVNCSessionMu.Lock()
	activeNoVNCSession = session
	noVNCSessionMu.Unlock()
}

func clearNoVNCSession(session *noVNCSession) {
	noVNCSessionMu.Lock()
	defer noVNCSessionMu.Unlock()
	if activeNoVNCSession == session {
		activeNoVNCSession = nil
	}
}

func startNoVNCSession(parent fyne.Window, opts *config.Options, onStateChanged func(bool)) {
	if !isVNCDisplaySelected(opts) {
		dialog.ShowInformation("VNC Required", "Select VNC display backend first.", parent)
		return
	}
	if isNoVNCSessionRunning() {
		dialog.ShowInformation("noVNC Already Running", "A noVNC session is already running. Stop it first.", parent)
		return
	}

	optsSnapshot := *opts
	host, tcpPort, err := vncTCPPort(&optsSnapshot)
	if err != nil {
		dialog.ShowError(err, parent)
		return
	}
	webPort, err := normalizePort(optsSnapshot.VNCWebPort, "6080")
	if err != nil {
		dialog.ShowError(err, parent)
		return
	}
	webListenHost := noVNCWebListenHost(&optsSnapshot)

	var progress *noVNCProgressDialog
	progress = newNoVNCProgressDialog(parent, func() {
		stopNoVNCSession()
		if onStateChanged != nil {
			fyne.Do(func() {
				onStateChanged(false)
			})
		}
		progress.SetStatus("noVNC stopped.")
		progress.AppendLog("Stop requested from progress dialog.")
		progress.SetStopEnabled(false)
	})
	progress.Show()
	progress.SetStatus("Preparing noVNC assets...")
	progress.SetProgress(0, 4)
	progress.AppendLog("Preparing noVNC assets (git clone/cache).")

	go func() {
		showError := func(err error) {
			progress.Close()
			fyne.Do(func() {
				if errors.Is(err, errGitNotFound) {
					showGitInstallHelperDialog(parent)
					return
				}
				showNoVNCStartupError(parent, err, "")
			})
		}

		noVNCDir, err := ensureNoVNCAssetsAvailable(context.Background(), func(line string) {
			progress.AppendLog(line)
		})
		if err != nil {
			showError(err)
			return
		}
		progress.SetProgress(1, 4)

		vncOpts := optsSnapshot
		vncOpts.Display = buildVNCDisplaySpec(&vncOpts)

		progress.SetStatus("Starting QEMU with VNC backend and socket forwarding...")
		progress.AppendLog("Starting QEMU VNC endpoint used by socket forwarder.")
		qemuCmd, qemuOutput, err := qemu.StartWithOutput(context.Background(), &vncOpts, qemuSystemBinaryPath())
		if err != nil {
			showError(err)
			return
		}
		qemuExitCh := make(chan error, 1)
		go func() {
			qemuExitCh <- qemuCmd.Wait()
			close(qemuExitCh)
		}()
		progress.SetProgress(2, 4)
		bridgeTargetHost := vncBridgeDialHost(host)
		bridgeTargetAddr := net.JoinHostPort(bridgeTargetHost, strconv.Itoa(tcpPort))
		progress.AppendLog(fmt.Sprintf("QEMU VNC configured at %s:%d", host, tcpPort))
		if bridgeTargetHost != host {
			progress.AppendLog(fmt.Sprintf("Bridge dial host normalized to %s for local connect.", bridgeTargetHost))
		}
		progress.AppendLog(fmt.Sprintf("Waiting for VNC target %s to accept TCP...", bridgeTargetAddr))
		if err := waitForTCPServer(bridgeTargetAddr, 12*time.Second, qemuExitCh); err != nil {
			_ = qemuCmd.Process.Kill()
			progress.Close()
			fyne.Do(func() {
				showNoVNCStartupError(parent, err, compactOutput(qemuOutput.String()))
			})
			return
		}
		progress.AppendLog("VNC target is reachable.")

		progress.SetStatus("Starting internal WebSocket bridge...")
		progress.AppendLog("Starting internal TCP <-> WebSocket forwarding.")
		server, err := startNoVNCServer(noVNCDir, webListenHost, webPort, bridgeTargetHost, tcpPort, func(line string) {
			progress.AppendLog(line)
		})
		if err != nil {
			_ = qemuCmd.Process.Kill()
			showError(err)
			return
		}
		progress.SetProgress(3, 4)

		session := &noVNCSession{qemuCmd: qemuCmd, server: server}
		setNoVNCSession(session)
		progress.SetStopEnabled(true)
		if onStateChanged != nil {
			fyne.Do(func() {
				onStateChanged(true)
			})
		}

		go func() {
			waitErr := <-qemuExitCh
			if waitErr != nil {
				progress.AppendLog(fmt.Sprintf("QEMU process exited: %v", waitErr))
			} else {
				progress.AppendLog("QEMU process exited.")
			}
			progress.AppendLog("noVNC web server is still running. Use 'Stop noVNC' to stop it.")
		}()

		progress.SetStatus("Opening noVNC in browser...")
		progress.AppendLog("Starting noVNC web server and opening browser.")
		noVNCURL := buildNoVNCURL(server.listenHost, server.listenPort)
		progress.AppendLog(fmt.Sprintf("Browser URL host/port: %s:%s", noVNCBrowserHost(server.listenHost), server.listenPort))
		progress.AppendLog("WebSocket path: /websockify")
		progress.SetPublicURL("")
		if lanIP := localLANIPv4(); lanIP != "" {
			lanURL := &url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort(lanIP, server.listenPort),
				Path:   "/vnc.html",
			}
			lanQuery := url.Values{}
			lanQuery.Set("autoconnect", "true")
			lanQuery.Set("resize", "scale")
			lanQuery.Set("host", lanIP)
			lanQuery.Set("port", server.listenPort)
			lanQuery.Set("path", "websockify")
			lanURL.RawQuery = lanQuery.Encode()
			progress.AppendLog(fmt.Sprintf("Local network endpoint: %s", lanURL.String()))
			progress.SetPublicURL(lanURL.String())
		}
		healthURL := fmt.Sprintf("http://%s:%s/vnc.html", noVNCBrowserHost(server.listenHost), server.listenPort)
		if err := waitForHTTPServer(healthURL, 5*time.Second); err != nil {
			stopNoVNCSession()
			if onStateChanged != nil {
				fyne.Do(func() {
					onStateChanged(false)
				})
			}
			progress.SetStopEnabled(false)
			progress.Close()
			fyne.Do(func() {
				showNoVNCStartupError(parent, fmt.Errorf("noVNC server did not become ready: %w", err), compactOutput(qemuOutput.String()))
			})
			return
		}
		progress.AppendLog("noVNC web server is reachable.")
		var openErr error
		fyne.DoAndWait(func() {
			openErr = fyne.CurrentApp().OpenURL(noVNCURL)
		})
		if openErr != nil {
			stopNoVNCSession()
			if onStateChanged != nil {
				fyne.Do(func() {
					onStateChanged(false)
				})
			}
			progress.SetStopEnabled(false)
			progress.Close()
			fyne.Do(func() {
				showNoVNCStartupError(parent, fmt.Errorf("failed to open browser: %w", openErr), compactOutput(qemuOutput.String()))
			})
			return
		}

		progress.SetProgress(4, 4)
		progress.SetStatus("noVNC is running.")
		progress.AppendLog("Done: noVNC started successfully.")
		progress.AppendLog(fmt.Sprintf("URL: %s", noVNCURL.String()))
		progress.AppendLog("Use the 'Stop noVNC' button in the main window to terminate this session.")
	}()
}
