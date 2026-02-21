package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/benworks/menuworks/config"
	"github.com/benworks/menuworks/menu"
)

// DrawMenu renders the current menu on screen
func (s *Screen) DrawMenu(navigator *menu.Navigator, disabledItems map[string]bool) {
	w, h := s.Size()

	// Center the menu in an 80x25 layout
	menuWidth := 60
	menuHeight := 18
	startX := (w - menuWidth) / 2
	startY := (h - menuHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	// Clear the area
	s.ClearRect(0, 0, w, h)

	// Fill menu interior with menu background color
	for dy := 0; dy < menuHeight; dy++ {
		for dx := 0; dx < menuWidth; dx++ {
			s.DrawChar(startX+dx, startY+dy, ' ', StyleMenuBg())
		}
	}

	// Draw menu frame with menu background for borders
	title := navigator.GetFormattedTitle()
	s.DrawBorderWithStyle(startX, startY, menuWidth, menuHeight, " "+title+" ", StyleBorderMenuBg())
	s.DrawShadow(startX, startY, menuWidth, menuHeight)

	// Draw header separator line with menu background
	headerSepY := startY + 2
	borderStyle := StyleBorderMenuBg()
	s.DrawBoxChar(startX, headerSepY, boxDoubleTLeft, borderStyle)
	s.DrawBoxChar(startX+menuWidth-1, headerSepY, boxDoubleTRight, borderStyle)
	for i := 1; i < menuWidth-1; i++ {
		s.DrawBoxChar(startX+i, headerSepY, boxDoubleHorizontal, borderStyle)
	}

	// Draw date/time inside title bar with menu background
	date := FormatDate()
	time := FormatTime()
	leftText := date + "     " + "Menu Works" // 5 spaces
	timeX := startX + menuWidth - 3 - len(time)
	s.DrawString(startX+2, startY+1, leftText, StyleTextMenuBg())
	s.DrawString(timeX, startY+1, time, StyleTextMenuBg())

	// Draw menu items
	items := navigator.GetCurrentMenu()
	selectedIdx := navigator.GetSelectionIndex()
	contentStartY := startY + 3
	maxItems := menuHeight - 4

	// Filter selectable items and draw them
	selectableCount := 0
	for _, item := range items {
		if item.Type == "separator" {
			continue
		}
		selectableCount++
	}

	// If no selectable items, show placeholder
	if selectableCount == 0 {
		s.drawEmptyMenuPlaceholder(startX, contentStartY, menuWidth, maxItems)
	} else {
		s.drawMenuItems(startX, contentStartY, menuWidth, maxItems, items, selectedIdx, navigator)
	}

	// Draw footer with helpful text
	footerY := startY + menuHeight + 1
	footerText := "↑↓/Scroll: Navigate | ENTER/Click: Select | ESC/RClick: Back | R: Reload | F2: Help"
	if footerY < h {
		s.DrawString(startX, footerY, footerText, StyleNormal())
	}

	s.HideCursor()
	s.Sync()
}

// DrawCommandOutput displays command output in a scrollable full-screen viewer
// Returns when user presses any key
func (s *Screen) DrawCommandOutput(output string, eventChan <-chan tcell.Event) {
	w, h := s.Size()

	// Split output into lines
	lines := strings.Split(output, "\n")

	// Track scrolling position
	scrollOffset := 0
	visibleLines := h - 3 // Space for header and footer

	for {
		s.ClearRect(0, 0, w, h)

		// Draw header
		headerText := "─ Command Output ─"
		headerX := (w - len(headerText)) / 2
		s.DrawString(headerX, 0, headerText, StyleBorder())

		// Draw visible lines
		for i := 0; i < visibleLines && scrollOffset+i < len(lines); i++ {
			line := lines[scrollOffset+i]
			// Truncate line to fit screen width
			if len(line) > w {
				line = line[:w]
			}
			s.DrawString(0, 1+i, line, StyleNormal())
		}

		// Draw footer with navigation info
		footerY := h - 1
		var footerText string
		if len(lines) <= visibleLines {
			footerText = "Press any key to return"
		} else {
			totalLines := len(lines)
			endLine := scrollOffset + visibleLines
			if endLine > totalLines {
				endLine = totalLines
			}
			footerText = fmt.Sprintf("Lines %d-%d of %d | ↑↓ or PgUp/PgDn to scroll", scrollOffset+1, endLine, totalLines)
		}
		footerX := (w - len(footerText)) / 2
		s.DrawString(footerX, footerY, footerText, StyleBorder())

		s.Sync()

		// Wait for input
		ev := <-eventChan
		if ev == nil {
			continue
		}

		keyEv, ok := ev.(*tcell.EventKey)
		if !ok {
			continue
		}

		// Handle navigation
		switch keyEv.Key() {
		case tcell.KeyUp:
			if scrollOffset > 0 {
				scrollOffset--
			}
		case tcell.KeyDown:
			if scrollOffset < len(lines)-visibleLines {
				scrollOffset++
			}
		case tcell.KeyPgUp:
			scrollOffset -= visibleLines
			if scrollOffset < 0 {
				scrollOffset = 0
			}
		case tcell.KeyPgDn:
			scrollOffset += visibleLines
			if scrollOffset > len(lines)-visibleLines {
				scrollOffset = len(lines) - visibleLines
			}
			if scrollOffset < 0 {
				scrollOffset = 0
			}
		default:
			// Any other key returns to menu
			return
		}
	}
}

// drawEmptyMenuPlaceholder draws the "(No items)" placeholder
func (s *Screen) drawEmptyMenuPlaceholder(x, y, width, height int) {
	placeholder := "(No items)"
	placeholderX := x + (width-len(placeholder))/2

	if placeholderY := y + height/2 - 1; placeholderY >= 0 {
		s.DrawString(placeholderX, placeholderY, placeholder, StyleTextMenuBg())
	}

	// Show Back/Quit option
	backText := "[B]ack"
	backX := x + (width-len(backText))/2
	if backY := y + height/2 + 1; backY >= 0 {
		s.DrawString(backX, backY, backText, StyleTextMenuBg())
	}
}

// drawMenuItems draws all menu items
func (s *Screen) drawMenuItems(x, y, width, maxItems int, items []config.MenuItem, selectedIdx int, navigator *menu.Navigator) {
	contentLineIdx := 0

	for i, item := range items {
		if contentLineIdx >= maxItems {
			break
		}

		if item.Type == "separator" {
			// Draw separator line with border color on menu background
			separatorY := y + contentLineIdx
			if separatorY >= 0 {
				for col := 1; col < width-1; col++ {
					s.DrawChar(x+col, separatorY, '─', StyleBorderMenuBg())
				}
			}
			contentLineIdx++
		} else {
			// Draw menu item
			itemY := y + contentLineIdx
			isSelected := (i == selectedIdx)
			isDisabled := navigator.IsItemDisabled(i)

			s.drawMenuItem(x, itemY, width, item, isSelected, isDisabled, navigator)
			contentLineIdx++
		}
	}
}

// drawMenuItem draws a single menu item
func (s *Screen) drawMenuItem(x, y, width int, item config.MenuItem, isSelected, isDisabled bool, navigator *menu.Navigator) {
	// Determine style for normal text
	var style tcell.Style
	var hotkeyStyle tcell.Style
	
	if isDisabled {
		style = StyleDisabledMenuBg()
		hotkeyStyle = StyleDisabledMenuBg()
	} else if isSelected {
		style = StyleHighlight()
		hotkeyStyle = StyleHotkeyHighlight()
	} else {
		style = StyleTextMenuBg()
		hotkeyStyle = StyleHotkeyMenuBg()
	}

	// Clear the line with menu background color
	s.ClearRectWithStyle(x+1, y, width-2, 1, StyleMenuBg())

	// Build the display text
	label := item.Label
	if len(label) > width-6 {
		label = TruncateString(label, width-6)
	}

	// Draw the item content
	itemContentX := x + 2
	itemContent := fmt.Sprintf(" %s ", label)

	// Get hotkey if applicable
	hotkey := item.Hotkey
	// Note: Auto-assigned hotkeys are handled in the menu package
	// If needed, add a public method to navigator to fetch hotkeys for display

	// Render text with potential hotkey highlighting
	currentX := itemContentX
	if isSelected && !isDisabled {
		// Render with hotkey highlighting in selected state
		currentX = s.drawItemWithHotkey(currentX, y, itemContent, hotkey, hotkeyStyle, style)
	} else {
		// Render with hotkey highlighting in normal/disabled state
		currentX = s.drawItemWithHotkey(currentX, y, itemContent, hotkey, hotkeyStyle, style)
	}

	// Draw menu item type indicator (► for submenu)
	if item.Type == "submenu" && !isDisabled {
		typeIndicatorX := (x + width - 3)
		if typeIndicatorX > currentX {
			typeStyle := StyleHighlight()
			if !isSelected {
				typeStyle = StyleBorderMenuBg()
			}
			s.DrawChar(typeIndicatorX, y, '►', typeStyle)
		}
	}
}

// drawItemWithHotkey draws the item text with hotkey highlighting
func (s *Screen) drawItemWithHotkey(x, y int, text, hotkey string, hotkeyStyle, normalStyle tcell.Style) int {
	currentX := x

	if hotkey == "" {
		// No hotkey, just draw the text
		segs := ParseHotkeyLabel(text, hotkey)
		for _, seg := range segs {
			currentX += s.DrawString(currentX, y, seg.Text, normalStyle)
		}
	} else {
		// Draw with hotkey highlighting
		hotkeyChar := rune(strings.ToUpper(hotkey)[0])
		for _, ch := range text {
			if ch == hotkeyChar {
				s.DrawChar(currentX, y, ch, hotkeyStyle)
			} else {
				s.DrawChar(currentX, y, ch, normalStyle)
			}
			currentX++
		}
	}

	return currentX
}

// indexOf returns the index of an item in a slice (helper for finding hotkey mapping)
func indexOf(items []config.MenuItem, target config.MenuItem) int {
	for i, item := range items {
		if item.Label == target.Label && item.Type == target.Type {
			return i
		}
	}
	return -1
}

// DrawDialog renders a dialog box with buttons
func (s *Screen) DrawDialog(title, message string, buttons []string, eventChan <-chan tcell.Event) int {
	w, h := s.Size()

	// Dialog size
	dialogWidth := 50
	dialogHeight := 12
	startX := (w - dialogWidth) / 2
	startY := (h - dialogHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	// Clear background
	s.ClearRect(0, 0, w, h)

	// Draw border
	s.DrawBorder(startX, startY, dialogWidth, dialogHeight, " "+title+" ")

	// Draw message text wrapped
	messageStartY := startY + 2
	lines := WrapText(message, dialogWidth-4)
	for i, line := range lines {
		if i >= 5 {
			break
		}
		msgX := startX + 2
		msgY := messageStartY + i
		if msgY < h {
			s.DrawString(msgX, msgY, line, StyleNormal())
		}
	}

	// Draw buttons
	buttonY := startY + dialogHeight - 3
	buttonSpacing := (dialogWidth - 4) / len(buttons)
	for i, btn := range buttons {
		btnX := startX + 2 + (i * buttonSpacing)
		btnText := fmt.Sprintf("[%s]", btn)
		if btnX+len(btnText) < startX+dialogWidth-1 {
			if buttonY < h {
				s.DrawString(btnX, buttonY, btnText, StyleHighlight())
			}
		}
	}

	s.Sync()

	// Simple event loop for button selection
	selectedButton := 0
	for {
		ev := <-eventChan
		switch e := ev.(type) {
		case *tcell.EventKey:
			switch e.Key() {
			case tcell.KeyLeft:
				selectedButton = (selectedButton - 1 + len(buttons)) % len(buttons)
			case tcell.KeyRight:
				selectedButton = (selectedButton + 1) % len(buttons)
			case tcell.KeyEnter:
				return selectedButton
			case tcell.KeyEscape:
				return 0 // Default to first button on ESC
			}

			// Redraw with new selection
			s.ClearRect(0, 0, w, h)
			s.DrawBorder(startX, startY, dialogWidth, dialogHeight, " "+title+" ")
			for i, line := range lines {
				if i >= 5 {
					break
				}
				msgX := startX + 2
				msgY := messageStartY + i
				if msgY < h {
					s.DrawString(msgX, msgY, line, StyleNormal())
				}
			}

			// Redraw buttons with selection
			for i, btn := range buttons {
				btnX := startX + 2 + (i * buttonSpacing)
				btnText := fmt.Sprintf("[%s]", btn)
				style := StyleHighlight()
				if i != selectedButton {
					style = StyleNormal()
				}
				if btnX+len(btnText) < startX+dialogWidth-1 {
					if buttonY < h {
						s.DrawString(btnX, buttonY, btnText, style)
					}
				}
			}
			s.Sync()
		}
	}
}

// WrapText wraps text to fit within maxWidth
func WrapText(text string, maxWidth int) []string {
	if maxWidth < 1 {
		maxWidth = 1
	}
	var lines []string
	// Split on explicit newlines first, then wrap each paragraph
	paragraphs := strings.Split(text, "\n")
	for _, para := range paragraphs {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		var currentLine string
		for _, word := range words {
			// If the word itself is longer than maxWidth, hard-break it
			for len(word) > maxWidth {
				if currentLine != "" {
					lines = append(lines, currentLine)
					currentLine = ""
				}
				lines = append(lines, word[:maxWidth])
				word = word[maxWidth:]
			}
			if len(word) == 0 {
				continue
			}
			if len(currentLine)+1+len(word) <= maxWidth {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			} else {
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = word
			}
		}
		if currentLine != "" {
			lines = append(lines, currentLine)
		}
	}
	return lines
}

// DrawSplashScreen renders the splash screen
func (s *Screen) DrawSplashScreen(version string) {
	w, h := s.Size()

	// Clear screen
	s.Clear()

	// Draw splash box
	splashWidth := 50
	splashHeight := 12
	startX := (w - splashWidth) / 2
	startY := (h - splashHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	s.DrawBorder(startX, startY, splashWidth, splashHeight, "")

	// Draw content
	titleY := startY + 3
	titleText := "MenuWorks 3.X"
	titleX := startX + (splashWidth-len(titleText))/2
	if titleY < h {
		s.DrawString(titleX, titleY, titleText, StyleHighlight())
	}

	versionY := startY + 5
	versionText := fmt.Sprintf("Version: %s", version)
	versionX := startX + (splashWidth-len(versionText))/2
	if versionY < h {
		s.DrawString(versionX, versionY, versionText, StyleNormal())
	}

	creditsY := startY + 7
	creditsText := "A Retro DOS-Style TUI"
	creditsX := startX + (splashWidth-len(creditsText))/2
	if creditsY < h {
		s.DrawString(creditsX, creditsY, creditsText, StyleNormal())
	}

	s.Sync()
}

// ShowItemHelp displays a dialog with command info and help text for a menu item
func (s *Screen) ShowItemHelp(command, help string, eventChan <-chan tcell.Event) {
	w, h := s.Size()

	// Dialog dimensions
	dialogWidth := 60
	dialogHeight := 14
	startX := (w - dialogWidth) / 2
	startY := (h - dialogHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	// Build the help dialog message
	var messageLines []string
	messageLines = append(messageLines, "Command:")
	messageLines = append(messageLines, command)

	// Add help text if available
	if help != "" {
		messageLines = append(messageLines, "")
		messageLines = append(messageLines, help)
	}

	// Render loop
	for {
		s.ClearRect(0, 0, w, h)
		s.DrawBorder(startX, startY, dialogWidth, dialogHeight, " Item Info ")

		// Draw message lines
		msgX := startX + 2
		msgY := startY + 2
		for _, line := range messageLines {
			if msgY >= startY+dialogHeight-3 {
				break
			}
			// Handle empty lines (blank space)
			if line == "" {
				msgY++
				continue
			}
			// Wrap text to fit dialog width
			wrappedLines := WrapText(line, dialogWidth-4)
			for _, wrappedLine := range wrappedLines {
				if msgY >= startY+dialogHeight-3 {
					break
				}
				if msgY < h {
					s.DrawString(msgX, msgY, wrappedLine, StyleNormal())
				}
				msgY++
			}
		}

		// Draw OK button
		buttonY := startY + dialogHeight - 2
		btnX := startX + (dialogWidth-len("[OK]"))/2 - 1
		if buttonY < h {
			s.DrawString(btnX, buttonY, "[OK]", StyleHighlight())
		}

		s.Sync()

		// Handle input
		ev := <-eventChan
		if keyEv, ok := ev.(*tcell.EventKey); ok {
			switch keyEv.Key() {
			case tcell.KeyEnter, tcell.KeyEscape:
				return
			}
		}
	}
}

