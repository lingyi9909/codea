// Package theme provides the Tokyo Night color palette and derived styles for
// the Codea TUI. It depends only on lipgloss and carries no Runtime or Vendor
// knowledge.
package theme

import "github.com/charmbracelet/lipgloss"

// Tokyo Night palette (design doc §4.2).
var (
	Primary    = lipgloss.Color("#c0caf5")
	Secondary  = lipgloss.Color("#a9b1d6")
	Muted      = lipgloss.Color("#565f89")
	Accent     = lipgloss.Color("#e0af68")
	Success    = lipgloss.Color("#9ece6a")
	Error      = lipgloss.Color("#f7768e")
	Border     = lipgloss.Color("#292e42")
	Background = lipgloss.Color("#1a1b26")
)

// ChatStyle is the default foreground style for chat text.
func ChatStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(Primary) }

// MutedStyle is a dim italic style for secondary/de-emphasized text.
func MutedStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(Muted).Italic(true) }

// AccentStyle highlights tool names and emphasis.
func AccentStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(Accent) }

// SuccessStyle renders success status text.
func SuccessStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(Success) }

// ErrorStyle renders error status text.
func ErrorStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(Error) }
