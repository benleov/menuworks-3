package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
)

// Screen wraps tcell screen with rendering utilities
type Screen struct {
	tcellScreen tcell.Screen
}

// NewScreen initializes and returns a new Screen
func NewScreen() (*Screen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	if err := s.Init(); err != nil {
		return nil, err
	}

	// Set color palette
	s.SetStyle(defaultStyle())

	return &Screen{tcellScreen: s}, nil
}

// Close closes the screen
func (s *Screen) Close() {
	s.tcellScreen.Fini()
}

// Size returns terminal width and height
func (s *Screen) Size() (width, height int) {
	return s.tcellScreen.Size()
}

// Clear clears the screen
func (s *Screen) Clear() {
	s.tcellScreen.Clear()
}

// ShowCursor shows the cursor
func (s *Screen) ShowCursor(x, y int) {
	s.tcellScreen.ShowCursor(x, y)
}

// HideCursor hides the cursor
func (s *Screen) HideCursor() {
	s.tcellScreen.HideCursor()
}

// Sync syncs the screen
func (s *Screen) Sync() {
	s.tcellScreen.Sync()
}

// PollEvent polls for an event
func (s *Screen) PollEvent() tcell.Event {
	return s.tcellScreen.PollEvent()
}

// StartEventPoller starts a goroutine that continuously polls for events
// and sends them to the returned channel. This prevents goroutine leaks.
func (s *Screen) StartEventPoller() <-chan tcell.Event {
	eventChan := make(chan tcell.Event)
	go func() {
		for {
			ev := s.tcellScreen.PollEvent()
			if ev == nil {
				return
			}
			eventChan <- ev
		}
	}()
	return eventChan
}

// SetCellUnsafe sets a cell at (x, y) with the given character and style
func (s *Screen) SetCellUnsafe(x, y int, r rune, st tcell.Style) {
	s.tcellScreen.SetCell(x, y, st, r)
}

// RefreshTheme updates the screen's default style to reflect current theme colors
func (s *Screen) RefreshTheme() {
	s.tcellScreen.SetStyle(defaultStyle())
}



// Color constants for VGA palette (mutable for theme support)
var (
	darkBlue     = tcell.ColorBlue
	brightCyan   = tcell.ColorAqua
	white        = tcell.ColorWhite
	lightGray    = tcell.Color250  // Light gray
	darkGray     = tcell.Color240  // Dark gray for shadow
	brightYellow = tcell.ColorYellow
	
	// Theme-specific colors (can be overridden by ApplyTheme)
	colorBackground  = tcell.ColorBlue
	colorText        = tcell.Color250
	colorBorder      = tcell.ColorAqua
	colorHighlightBg = tcell.ColorBlue
	colorHighlightFg = tcell.ColorWhite
	colorHotkey      = tcell.ColorYellow
	colorShadow      = tcell.Color240
	colorDisabled    = tcell.Color240
)

// ThemeColors represents a color scheme for the UI
type ThemeColors struct {
	Background  string
	Text        string
	Border      string
	HighlightBg string
	HighlightFg string
	Hotkey      string
	Shadow      string
	Disabled    string
}

// ApplyTheme updates the global color variables with the provided theme
// colorParser is a function that converts a color name to tcell.Color
func ApplyTheme(theme ThemeColors, colorParser func(string) (tcell.Color, bool)) {
	// Helper to apply color or keep default
	applyColor := func(colorName string, defaultColor tcell.Color) tcell.Color {
		if color, valid := colorParser(colorName); valid {
			return color
		}
		return defaultColor
	}
	
	// Update theme-specific colors
	colorBackground = applyColor(theme.Background, tcell.ColorBlue)
	colorText = applyColor(theme.Text, tcell.Color250)
	colorBorder = applyColor(theme.Border, tcell.ColorAqua)
	colorHighlightBg = applyColor(theme.HighlightBg, tcell.ColorBlue)
	colorHighlightFg = applyColor(theme.HighlightFg, tcell.ColorWhite)
	colorHotkey = applyColor(theme.Hotkey, tcell.ColorYellow)
	colorShadow = applyColor(theme.Shadow, tcell.Color240)
	colorDisabled = applyColor(theme.Disabled, tcell.Color240)
	
	// Update legacy color variables for backwards compatibility
	darkBlue = colorBackground
	brightCyan = colorBorder
	white = colorHighlightFg
	lightGray = colorText
	darkGray = colorShadow
	brightYellow = colorHotkey
}

// defaultStyle returns the default style (uses theme colors)
func defaultStyle() tcell.Style {
	return tcell.StyleDefault.
		Foreground(colorText).
		Background(colorBackground)
}

// StyleNormal returns the normal style (uses theme colors)
func StyleNormal() tcell.Style {
	return tcell.StyleDefault.
		Foreground(colorText).
		Background(colorBackground)
}

// StyleBorder returns the border style (uses theme colors)
func StyleBorder() tcell.Style {
	return tcell.StyleDefault.
		Foreground(colorBorder).
		Background(colorBackground)
}

// StyleHighlight returns the highlight style (uses theme colors)
func StyleHighlight() tcell.Style {
	return tcell.StyleDefault.
		Foreground(colorHighlightFg).
		Background(colorHighlightBg)
}

// StyleShadow returns the shadow style (uses theme colors)
func StyleShadow() tcell.Style {
	return tcell.StyleDefault.
		Foreground(colorShadow).
		Background(colorShadow)
}

// StyleHotkey returns the hotkey style (uses theme colors)
func StyleHotkey() tcell.Style {
	return tcell.StyleDefault.
		Foreground(colorHotkey).
		Background(colorBackground).
		Bold(true)
}

// StyleHotkeyHighlight returns the hotkey highlight style (uses theme colors)
func StyleHotkeyHighlight() tcell.Style {
	return tcell.StyleDefault.
		Foreground(colorHotkey).
		Background(colorHighlightBg).
		Bold(true)
}

// StyleDisabled returns the disabled style (uses theme colors)
func StyleDisabled() tcell.Style {
	return tcell.StyleDefault.
		Foreground(colorDisabled).
		Background(colorBackground)
}

// FormatDate returns current date in DD/MM/YY format
func FormatDate() string {
	now := time.Now()
	return now.Format("02/01/06")
}

// FormatTime returns current time in H:MM AM/PM format (uppercase, no leading zero on hour)
func FormatTime() string {
	now := time.Now()
	hour := now.Hour()
	minute := now.Minute()
	ampm := "AM"
	
	if hour >= 12 {
		ampm = "PM"
		if hour > 12 {
			hour -= 12
		}
	}
	if hour == 0 {
		hour = 12
	}
	
	return fmt.Sprintf("%d:%02d %s", hour, minute, ampm)
}

// DrawBoxChar draws a UTF-8 box character at (x, y)
func (s *Screen) DrawBoxChar(x, y int, ch rune, style tcell.Style) {
	if x < 0 || y < 0 {
		return
	}
	w, h := s.Size()
	if x >= w || y >= h {
		return
	}
	s.SetCellUnsafe(x, y, ch, style)
}

// Box-drawing characters (UTF-8 double-line)
const (
	boxDoubleHorizontal = '═'
	boxDoubleVertical   = '║'
	boxDoubleTopLeft    = '╔'
	boxDoubleTopRight   = '╗'
	boxDoubleBottomLeft = '╚'
	boxDoubleBottomRight = '╝'
	boxDoubleCross      = '╬'
	boxDoubleTDown      = '╦'
	boxDoubleTUp         = '╩'
	boxDoubleTRight      = '╣'
	boxDoubleTLeft       = '╠'
)

// Shadow character
const shadowChar = ' '

// DrawChar draws a character at (x, y) with style
func (s *Screen) DrawChar(x, y int, ch rune, style tcell.Style) {
	if x < 0 || y < 0 {
		return
	}
	w, h := s.Size()
	if x >= w || y >= h {
		return
	}
	s.SetCellUnsafe(x, y, ch, style)
}

// DrawString draws a string starting at (x, y) with style, truncating if needed
func (s *Screen) DrawString(x, y int, text string, style tcell.Style) int {
	w, h := s.Size()
	if y < 0 || y >= h || x >= w {
		return 0
	}

	colsWritten := 0
	for _, ch := range text {
		if x+colsWritten >= w {
			break
		}
		s.DrawChar(x+colsWritten, y, ch, style)
		colsWritten++
	}
	return colsWritten
}

// TruncateString truncates a string to fit within maxWidth, adding ellipsis if needed
func TruncateString(text string, maxWidth int) string {
	if len(text) <= maxWidth {
		return text
	}
	if maxWidth <= 0 {
		return ""
	}
	if maxWidth < 3 {
		return text[:maxWidth]
	}
	return text[:maxWidth-1] + "…"
}

// HighlightHotkey returns the label with hotkey highlighted using ANSI-like markers
// This is a helper to structure text for proper display with hotkey styling
type HotkeylabelSegment struct {
	Text  string
	IsHotkey bool
}

// ParseHotkeyLabel parses a label and identifies the hotkey character position
func ParseHotkeyLabel(label, hotkey string) []HotkeylabelSegment {
	if hotkey == "" {
		return []HotkeylabelSegment{{Text: label, IsHotkey: false}}
	}

	hotkeyChar := rune(hotkey[0])
	var segments []HotkeylabelSegment
	found := false

	for _, ch := range label {
		if !found && ch == hotkeyChar {
			segments = append(segments, HotkeylabelSegment{
				Text:     string(ch),
				IsHotkey: true,
			})
			found = true
		} else {
			if len(segments) > 0 && !segments[len(segments)-1].IsHotkey {
				segments[len(segments)-1].Text += string(ch)
			} else {
				segments = append(segments, HotkeylabelSegment{
					Text:     string(ch),
					IsHotkey: false,
				})
			}
		}
	}

	return segments
}

// DrawBorder draws a double-line border box with optional title
func (s *Screen) DrawBorder(x, y, width, height int, title string) {
	w, h := s.Size()

	// Ensure bounds
	if x < 0 || y < 0 || width <= 0 || height <= 0 {
		return
	}

	borderStyle := StyleBorder()

	// Top-left corner
	if x < w && y < h {
		s.DrawBoxChar(x, y, boxDoubleTopLeft, borderStyle)
	}

	// Top-right corner
	if x+width-1 < w && y < h {
		s.DrawBoxChar(x+width-1, y, boxDoubleTopRight, borderStyle)
	}

	// Bottom-left corner
	if x < w && y+height-1 < h {
		s.DrawBoxChar(x, y+height-1, boxDoubleBottomLeft, borderStyle)
	}

	// Bottom-right corner
	if x+width-1 < w && y+height-1 < h {
		s.DrawBoxChar(x+width-1, y+height-1, boxDoubleBottomRight, borderStyle)
	}

	// Top and bottom horizontal lines
	for i := 1; i < width-1; i++ {
		if x+i < w {
			if y < h {
				s.DrawBoxChar(x+i, y, boxDoubleHorizontal, borderStyle)
			}
			if y+height-1 < h {
				s.DrawBoxChar(x+i, y+height-1, boxDoubleHorizontal, borderStyle)
			}
		}
	}

	// Left and right vertical lines
	for j := 1; j < height-1; j++ {
		if y+j < h {
			if x < w {
				s.DrawBoxChar(x, y+j, boxDoubleVertical, borderStyle)
			}
			if x+width-1 < w {
				s.DrawBoxChar(x+width-1, y+j, boxDoubleVertical, borderStyle)
			}
		}
	}

	// Draw title if provided
	if title != "" {
		titleX := x + 2
		titleLen := len(title)
		if titleLen > width-4 {
			titleLen = width - 4
			title = TruncateString(title, titleLen)
		}

		for i, ch := range title {
			if titleX+i < w && y < h {
				s.DrawBoxChar(titleX+i, y, ch, borderStyle)
			}
		}
	}
}

// DrawShadow draws a drop shadow effect (space char with dark gray background)
// Shadows are +1 row, +2 columns offset and clipped at terminal boundaries
func (s *Screen) DrawShadow(x, y, width, height int) {
	w, h := s.Size()

	// Right edge shadow
	shadowX := x + width + 1
	for j := y + 1; j < y+height+1; j++ {
		if shadowX < w && j < h {
			s.DrawChar(shadowX, j, shadowChar, StyleShadow())
		}
	}

	// Bottom edge shadow
	shadowY := y + height
	for i := x + 2; i < x+width+2; i++ {
		if i < w && shadowY < h {
			s.DrawChar(i, shadowY, shadowChar, StyleShadow())
		}
	}

	// Corner shadow
	if shadowX < w && shadowY < h {
		s.DrawChar(shadowX, shadowY, shadowChar, StyleShadow())
	}
}

// ClearRect clears a rectangular area
func (s *Screen) ClearRect(x, y, width, height int) {
	w, h := s.Size()
	defaultSt := StyleNormal()

	for j := y; j < y+height; j++ {
		if j < 0 || j >= h {
			continue
		}
		for i := x; i < x+width; i++ {
			if i < 0 || i >= w {
				continue
			}
			s.DrawChar(i, j, ' ', defaultSt)
		}
	}
}
