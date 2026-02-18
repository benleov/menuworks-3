package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/benworks/menuworks/config"
	"github.com/benworks/menuworks/exec"
	"github.com/benworks/menuworks/menu"
	"github.com/benworks/menuworks/ui"
)

var version = "1.0.0"

func main() {
	// Get binary directory for config file
	ex, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to determine executable path: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(filepath.Dir(ex), "config.yaml")

	// Initialize screen
	screen, err := ui.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize screen: %v\n", err)
		os.Exit(1)
	}
	defer screen.Close()

	// Start event poller IMMEDIATELY after screen init (needed by all functions)
	eventChan := screen.StartEventPoller()

	// Check terminal size and show resize loop if needed
	ensureTerminalSize(screen, eventChan)

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		handleConfigError(screen, eventChan, configPath, err)
		// If handleConfigError didn't exit, assume we should retry
		cfg, err = config.Load(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal: failed to load config: %v\n", err)
			os.Exit(1)
		}
	}

	// Show splash screen with fixed 400ms delay
	screen.DrawSplashScreen(version)
	
	// Consume and discard all events during splash (prevents macOS hang)
	// Per spec: "key events are consumed and discarded by reading and ignoring tcell events"
	splashStart := time.Now()
	for time.Since(splashStart) < 400*time.Millisecond {
		select {
		case <-eventChan:
			// Event discarded (consumed but ignored)
		case <-time.After(10 * time.Millisecond):
			// No event, continue waiting
		}
	}
	
	// Explicitly clear screen before transitioning to menu
	screen.Clear()
	screen.Sync()

	// Create navigator
	navigator := menu.NewNavigator(cfg)

	// Check for missing submenu targets on startup and report once per session
	checkAndReportMissingTargets(screen, navigator)

	// Main event loop
	mainLoop(screen, configPath, navigator, cfg, eventChan)
}

// ensureTerminalSize verifies terminal is at least 80x25 and loops until resized if too small
func ensureTerminalSize(screen *ui.Screen, eventChan <-chan tcell.Event) {
	for {
		w, h := screen.Size()
		if w >= 80 && h >= 25 {
			return // Terminal is large enough, proceed
		}

		// Draw error pop-up
		screen.Clear()
		dialogWidth := 50
		dialogHeight := 8
		startX := (w - dialogWidth) / 2
		if startX < 0 {
			startX = 0
		}
		startY := (h - dialogHeight) / 2
		if startY < 0 {
			startY = 0
		}

		screen.DrawBorder(startX, startY, dialogWidth, dialogHeight, " Terminal Too Small ")

		// Draw message
		msg := "Please resize your terminal to at least 80×25"
		msgX := startX + (dialogWidth - len(msg)) / 2
		if msgX < 0 {
			msgX = 0
		}
		msgY := startY + 2
		for i, ch := range msg {
			screen.DrawChar(msgX+i, msgY, ch, ui.StyleNormal())
		}

		msg2 := fmt.Sprintf("Current size: %d×%d", w, h)
		msg2X := startX + (dialogWidth - len(msg2)) / 2
		if msg2X < 0 {
			msg2X = 0
		}
		screen.DrawChar(msg2X, msgY+2, ' ', ui.StyleNormal())
		for i, ch := range msg2 {
			screen.DrawChar(msg2X+i, msgY+2, ch, ui.StyleNormal())
		}

		screen.Sync()

		// Wait for resize or other events
		ev := <-eventChan
		if ev != nil {
			// Check if Escape key was pressed
			if keyEv, ok := ev.(*tcell.EventKey); ok {
				if keyEv.Key() == tcell.KeyEscape {
					screen.Close()
					os.Exit(0)
				}
			}
			// Otherwise, discard event and loop to check size again
			continue
		}
	}
}

// checkTerminalSize verifies terminal is at least 80x25
func checkTerminalSize(screen *ui.Screen) error {
	w, h := screen.Size()
	if w < 80 || h < 25 {
		return fmt.Errorf("terminal too small (minimum 80x25, got %dx%d)", w, h)
	}
	return nil
}

// handleConfigError shows a dialog for config errors
func handleConfigError(screen *ui.Screen, eventChan <-chan tcell.Event, configPath string, err error) {
	w, h := screen.Size()

	// Ensure screen is large enough
	if w < 80 || h < 25 {
		fmt.Fprintf(os.Stderr, "Terminal too small for error dialog and cannot load config\n")
		os.Exit(1)
	}

	// Show error dialog with three options
	dialogWidth := 60
	dialogHeight := 14
	startX := (w - dialogWidth) / 2
	startY := (h - dialogHeight) / 2

	selectedBtn := 0

	for {
		screen.ClearRect(0, 0, w, h)
		screen.DrawBorder(startX, startY, dialogWidth, dialogHeight, " Config Error ")

		// Draw error message
		lines := []string{
			"Failed to load configuration.",
			fmt.Sprintf("Error: %v", err),
		}
		msgY := startY + 2
		for i, line := range lines {
			if i >= 5 {
				break
			}
			if msgY+i < h {
				screen.DrawString(startX+2, msgY+i, line, ui.StyleNormal())
			}
		}

		// Draw buttons
		buttons := []string{"Retry", "Use Default", "Exit"}
		buttonY := startY + dialogHeight - 3
		buttonSpacing := (dialogWidth - 4) / len(buttons)

		for i, btn := range buttons {
			btnX := startX + 2 + (i * buttonSpacing)
			btnText := fmt.Sprintf("[%s]", btn)
			style := ui.StyleNormal()
			if i == selectedBtn {
				style = ui.StyleHighlight()
			}
			if btnX+len(btnText) < startX+dialogWidth-1 {
				if buttonY < h {
					screen.DrawString(btnX, buttonY, btnText, style)
				}
			}
		}

		screen.Sync()

		// Handle input
		ev := <-eventChan
		if keyEv, ok := ev.(*tcell.EventKey); ok {
			switch keyEv.Key() {
			case tcell.KeyLeft:
				selectedBtn = (selectedBtn - 1 + len(buttons)) % len(buttons)
			case tcell.KeyRight:
				selectedBtn = (selectedBtn + 1) % len(buttons)
			case tcell.KeyEnter:
				switch selectedBtn {
				case 0: // Retry
					return
				case 1: // Use Default
					if err := config.WriteDefault(configPath); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to write default config: %v\n", err)
						os.Exit(1)
					}
					return
				case 2: // Exit
					os.Exit(0)
				}
			case tcell.KeyEscape:
				return
			}
		}
	}
}

// checkAndReportMissingTargets checks for missing submenu targets and reports them
func checkAndReportMissingTargets(screen *ui.Screen, navigator *menu.Navigator) {
	// Missing target errors will be reported per-menu the first time they're encountered
	// This is handled dynamically in the main event loop
}

// showErrorDialog shows a single-button error dialog
func showErrorDialog(screen *ui.Screen, eventChan <-chan tcell.Event, title, message string) {
	w, h := screen.Size()

	dialogWidth := 50
	dialogHeight := 11
	startX := (w - dialogWidth) / 2
	startY := (h - dialogHeight) / 2

	for {
		screen.ClearRect(0, 0, w, h)
		screen.DrawBorder(startX, startY, dialogWidth, dialogHeight, " "+title+" ")

		// Draw message
		lines := strings.Split(message, "\n")
		msgY := startY + 2
		for i, line := range lines {
			if i >= 5 {
				break
			}
			if msgY+i < h {
				screen.DrawString(startX+2, msgY+i, line, ui.StyleNormal())
			}
		}

		// Draw button
		buttonY := startY + dialogHeight - 2
		btnX := startX + (dialogWidth-len("[OK]"))/2 - 1
		if buttonY < h {
			screen.DrawString(btnX, buttonY, "[OK]", ui.StyleHighlight())
		}

		screen.Sync()

		// Handle input
		ev := <-eventChan
		if _, ok := ev.(*tcell.EventKey); ok {
			break
		}
	}
}

// mainLoop handles the main event loop
func mainLoop(screen *ui.Screen, configPath string, navigator *menu.Navigator, cfg *config.Config, eventChan <-chan tcell.Event) {
	handleSelection := func() {
		item, _ := navigator.GetSelectedItem()
		if item.Type == "submenu" {
			if err := navigator.Open(); err != nil {
				if !navigator.IsTargetErrorReported(navigator.GetCurrentMenuName()) {
					showErrorDialog(screen, eventChan, "Error", fmt.Sprintf("Error: %v", err))
					navigator.MarkTargetErrorReported(navigator.GetCurrentMenuName())
				}
			}
			return
		}

		if item.Type == "command" {
			// Determine if we should show output
			showOutput := true // Default
			if item.ShowOutput != nil {
				showOutput = *item.ShowOutput
			}

			// Get the command for the current OS
			command := item.Exec.CommandForOS(exec.GetOS())

			// Execute command and capture output
			output := exec.ExecuteAndCapture(command, item.Exec.WorkDir)

			if showOutput && output != "" {
				// Display output in scrollable viewer
				screen.DrawCommandOutput(output, eventChan)
			} else {
				// No output or user chose to hide output
				showMessageDialog(screen, eventChan, "Command Executed", "Command finished successfully.")
			}
			return
		}

		if item.Type == "back" {
			if navigator.IsAtRoot() {
				return
			}
			navigator.Back()
		}
	}

	// Main event loop
	for {
		// Check terminal size
		w, h := screen.Size()
		if w < 80 || h < 25 {
			showResizeError(screen)
			if err := waitForResize(screen, eventChan); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return
			}
			// Reload config after resize
			newCfg, err := config.Load(configPath)
			if err == nil {
				cfg = newCfg
				navigator = menu.NewNavigator(cfg)
			}
			continue
		}

		// Draw current menu
		disabledItems := make(map[string]bool) // Placeholder for now
		screen.DrawMenu(navigator, disabledItems)

		// Get event from poller channel
		ev := <-eventChan
		if ev == nil {
			continue
		}

		switch e := ev.(type) {
		case *tcell.EventKey:
			switch e.Key() {
			case tcell.KeyUp:
				navigator.PrevSelectable()

			case tcell.KeyDown:
				navigator.NextSelectable()

			case tcell.KeyRight, tcell.KeyEnter:
				handleSelection()

			case tcell.KeyLeft, tcell.KeyEscape:
				if navigator.IsAtRoot() {
					return // Exit
				}
				navigator.Back()

			case tcell.KeyRune:
				if e.Rune() == 'R' || e.Rune() == 'r' {
					// Reload config
					newCfg, err := config.Load(configPath)
					if err != nil {
						showErrorDialog(screen, eventChan, "Reload Error", fmt.Sprintf("Failed to reload config: %v", err))
					} else {
						cfg = newCfg
						// Preserve selection state as much as possible
						oldNavState := navigator.RememberSelection()

						navigator = menu.NewNavigator(cfg)
						navigator.RecallSelection(oldNavState)

						showMessageDialog(screen, eventChan, "Config Reloaded", "Configuration reloaded successfully.")
					}
					break
				}

				idx := navigator.SelectItemByHotkey(string(e.Rune()))
				if idx >= 0 {
					navigator.SetSelectionIndex(idx)
					handleSelection()
				}
			}

		case *tcell.EventResize:
			// Just re-render on resize
			continue
		}
	}
}

// showResizeError shows an error when terminal is too small
func showResizeError(screen *ui.Screen) {
	w, h := screen.Size()

	if w >= 80 && h >= 25 {
		return // No error if big enough
	}

	// Show error in small terminal
	fmt.Printf("Terminal too small (%dx%d). Minimum required: 80x25\n", w, h)
	fmt.Println("Resize your terminal and try again.")
}

// waitForResize waits for terminal to be resized to at least 80x25
func waitForResize(screen *ui.Screen, eventChan <-chan tcell.Event) error {
	for {
		ev := <-eventChan
		if _, ok := ev.(*tcell.EventResize); ok {
			w, h := screen.Size()
			if w >= 80 && h >= 25 {
				return nil
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// showMessageDialog shows a message dialog
func showMessageDialog(screen *ui.Screen, eventChan <-chan tcell.Event, title, message string) {
	w, h := screen.Size()

	dialogWidth := 50
	dialogHeight := 10
	startX := (w - dialogWidth) / 2
	startY := (h - dialogHeight) / 2

	for {
		screen.ClearRect(0, 0, w, h)
		screen.DrawBorder(startX, startY, dialogWidth, dialogHeight, " "+title+" ")

		// Draw message
		lines := strings.Split(message, "\n")
		msgY := startY + 2
		for i, line := range lines {
			if i >= 4 {
				break
			}
			if msgY+i < h {
				screen.DrawString(startX+2, msgY+i, line, ui.StyleNormal())
			}
		}

		// Draw button
		buttonY := startY + dialogHeight - 2
		btnX := startX + (dialogWidth-len("[OK]"))/2 - 1
		if buttonY < h {
			screen.DrawString(btnX, buttonY, "[OK]", ui.StyleHighlight())
		}

		screen.Sync()

		// Handle input
		ev := <-eventChan
		if _, ok := ev.(*tcell.EventKey); ok {
			break
		}
	}
}
