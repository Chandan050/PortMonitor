package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

type PortInfo struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	PID         int    `json:"pid"`
	ProcessName string `json:"processName"`
	Status      string `json:"status"`
}

type KillResult struct {
	Success       bool   `json:"success"`
	RequiresAdmin bool   `json:"requiresAdmin"`
	Message       string `json:"message"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ListPorts discovers all TCP LISTENING entries using Windows netstat.
// It deliberately does not require administrator rights for the normal scan.
func newHiddenCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

func (a *App) ListPorts() ([]PortInfo, error) {
	cmd := newHiddenCommand("netstat", "-ano", "-p", "tcp")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("netstat failed: %w", err)
	}

	// Cache process names so repeated PIDs do not cause repeated tasklist calls.
	processNames := make(map[int]string)
	ports := make(map[string]PortInfo)

	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || !strings.HasPrefix(strings.ToUpper(line), "TCP") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		state := strings.ToUpper(fields[3])
		if state != "LISTENING" {
			continue
		}

		local := fields[1]
		pid, err := strconv.Atoi(fields[4])
		if err != nil || pid <= 0 {
			continue
		}

		port, err := extractPort(local)
		if err != nil || port <= 0 || !isRegisteredPort(port) {
			continue
		}

		key := fmt.Sprintf("%d/%d", pid, port)
		if _, exists := ports[key]; exists {
			continue
		}

		name := "Unknown"
		if cached, ok := processNames[pid]; ok {
			name = cached
		} else {
			name = processName(pid)
			processNames[pid] = name
		}

		ports[key] = PortInfo{
			Port:        port,
			Protocol:    "TCP",
			PID:         pid,
			ProcessName: name,
			Status:      "Listening",
		}
	}

	result := make([]PortInfo, 0, len(ports))
	for _, p := range ports {
		result = append(result, p)
	}

	// Stable ordering: port, then PID.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Port < result[i].Port ||
				(result[j].Port == result[i].Port && result[j].PID < result[i].PID) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

func isRegisteredPort(port int) bool {
	return port >= 1024 && port <= 49151
}

func extractPort(endpoint string) (int, error) {
	// Handles:
	// 0.0.0.0:8080
	// 127.0.0.1:9000
	// [::]:3000
	// [::1]:5000
	idx := strings.LastIndex(endpoint, ":")
	if idx < 0 || idx == len(endpoint)-1 {
		return 0, fmt.Errorf("invalid endpoint")
	}
	return strconv.Atoi(endpoint[idx+1:])
}

func processName(pid int) string {
	cmd := newHiddenCommand("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	out, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}

	line := strings.TrimSpace(string(out))
	if line == "" || strings.Contains(strings.ToLower(line), "no tasks") {
		return "Unknown"
	}

	// CSV first field is the image name.
	if strings.HasPrefix(line, `"`) {
		end := strings.Index(line[1:], `"`)
		if end >= 0 {
			return line[1 : end+1]
		}
	}

	fields := strings.Split(line, ",")
	if len(fields) > 0 {
		return strings.Trim(fields[0], `"`)
	}
	return "Unknown"
}

func (a *App) KillProcess(pid int, processName string, port int) KillResult {
	if pid <= 0 {
		return KillResult{Message: "Invalid process ID."}
	}

	// Re-check that the process still exists before asking for confirmation.
	if !processExists(pid) {
		return KillResult{Message: fmt.Sprintf("Process %d is no longer running. The port may already be free.", pid)}
	}

	// Ask Windows to terminate the process. taskkill /F may require elevation.
	cmd := newHiddenCommand("taskkill", "/PID", strconv.Itoa(pid), "/F")
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err == nil {
		return KillResult{
			Success: true,
			Message: fmt.Sprintf("%s (PID %d) was terminated. Port %d will be refreshed.", processName, pid, port),
		}
	}

	lower := strings.ToLower(output)
	if strings.Contains(lower, "access is denied") ||
		strings.Contains(lower, "access denied") ||
		strings.Contains(lower, "requires elevation") ||
		strings.Contains(lower, "elevation") {
		return KillResult{
			RequiresAdmin: true,
			Message:       "Windows denied the termination because administrator privileges are required.",
		}
	}

	if strings.Contains(lower, "not found") ||
		strings.Contains(lower, "no running instance") {
		return KillResult{
			Success: true,
			Message: fmt.Sprintf("Process %d has already terminated.", pid),
		}
	}

	if output == "" {
		output = err.Error()
	}

	return KillResult{
		Message: fmt.Sprintf("Windows could not terminate PID %d: %s", pid, output),
	}
}

func processExists(pid int) bool {
	cmd := newHiddenCommand("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	text := strings.ToLower(string(out))
	return !strings.Contains(text, "no tasks")
}

// RelaunchElevated uses the Windows ShellExecute "runas" verb.
// No password is collected or stored by this application.
func (a *App) RelaunchAsAdministrator() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	workingDir, _ := os.Getwd()
	if workingDir == "" {
		workingDir = filepath.Dir(exe)
	}

	// Start-Process -Verb RunAs delegates authentication to Windows UAC.
	// No password is read by this application.
	ps := fmt.Sprintf(
		"Start-Process -FilePath '%s' -WorkingDirectory '%s' -Verb RunAs",
		powershellQuote(exe),
		powershellQuote(workingDir),
	)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not request administrator elevation: %w", err)
	}

	// Close the current non-elevated instance after the new one is requested.
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
	return nil
}

func powershellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func (a *App) CurrentTime() string {
	return time.Now().Format("15:04:05")
}
