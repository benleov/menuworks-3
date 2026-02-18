package exec

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/benworks/menuworks/ui"
)

// GetOS returns the current OS type string
func GetOS() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	case "darwin":
		return "darwin"
	default:
		return runtime.GOOS
	}
}

// Execute runs a command using the platform-appropriate shell
func Execute(command, workDir string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", command)
	default:
		cmd = exec.Command("sh", "-c", command)
	}

	// Inherit stdio/stdout/stderr so commands display naturally
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if resolvedDir := resolveWorkDir(command, workDir); resolvedDir != "" {
		cmd.Dir = resolvedDir
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// ExecuteAndCapture runs a command and captures its output
// Returns the combined stdout+stderr as a string
func ExecuteAndCapture(command, workDir string) string {
	var cmd *exec.Cmd
	var output bytes.Buffer

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", command)
	default:
		cmd = exec.Command("sh", "-c", command)
	}

	if resolvedDir := resolveWorkDir(command, workDir); resolvedDir != "" {
		cmd.Dir = resolvedDir
	}

	// Capture both stdout and stderr
	cmd.Stdout = &output
	cmd.Stderr = &output

	// Run the command, ignore errors (user will see output anyway)
	_ = cmd.Run()

	// Split output into lines and return
	result := strings.TrimSpace(output.String())
	return result
}
// showing the output, then prompts to return
func ExecuteInAltScreen(screen *ui.Screen, command, workDir string) error {
	// Close current screen to release tcell
	screen.Close()

	// Enable alternate screen buffer
	altScreen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to create alternate screen: %w", err)
	}
	if err := altScreen.Init(); err != nil {
		return fmt.Errorf("failed to init alternate screen: %w", err)
	}
	defer altScreen.Fini()

	// Clear alt screen
	altScreen.Clear()
	altScreen.Sync()

	// Execute the command with inherited I/O (shows output)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", command)
	default:
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if resolvedDir := resolveWorkDir(command, workDir); resolvedDir != "" {
		cmd.Dir = resolvedDir
	}

	_ = cmd.Run() // Run command, ignore errors for now (user sees output anyway)

	// Print prompt
	fmt.Println("\nCommand finished. Press any key to return.")

	// Wait for any key press
	for {
		ev := altScreen.PollEvent()
		if _, ok := ev.(*tcell.EventKey); ok {
			break
		}
	}

	altScreen.Fini()

	// Reinitialize tcell screen for menu
	newScreen, err := ui.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to restore screen: %w", err)
	}

	// Copy screen pointer back
	*screen = *newScreen

	return nil
}

func resolveWorkDir(command, workDir string) string {
	if strings.TrimSpace(workDir) != "" {
		return workDir
	}

	cmdPath := firstCommandToken(command)
	if cmdPath == "" {
		return ""
	}

	if _, err := os.Stat(cmdPath); err == nil {
		return filepath.Dir(cmdPath)
	}

	return ""
}

func firstCommandToken(command string) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "\"") {
		endIdx := strings.Index(trimmed[1:], "\"")
		if endIdx == -1 {
			return ""
		}
		return trimmed[1 : 1+endIdx]
	}

	for i, ch := range trimmed {
		if ch == ' ' || ch == '\t' || ch == '\n' {
			return trimmed[:i]
		}
	}

	return trimmed
}
