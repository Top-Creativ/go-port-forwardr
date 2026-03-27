package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Palette
	colorBg      = lipgloss.Color("#0f1117")
	colorSurface = lipgloss.Color("#1a1d27")
	colorBorder  = lipgloss.Color("#2e3147")
	colorAccent  = lipgloss.Color("#7c6af7")
	colorAccent2 = lipgloss.Color("#55d4c8")
	colorMuted   = lipgloss.Color("#6b7280")
	colorText    = lipgloss.Color("#e2e8f0")
	colorSuccess = lipgloss.Color("#34d399")
	colorError   = lipgloss.Color("#f87171")
	colorWarning = lipgloss.Color("#fbbf24")

	// Base
	StyleBase = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorText)

	// Header banner
	StyleBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 2).
			Margin(0, 0, 1, 0)

	// Cards / panels
	StyleCard = lipgloss.NewStyle().
			Background(colorSurface).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			Margin(0, 0, 1, 0)

	// Selected row
	StyleSelected = lipgloss.NewStyle().
			Background(colorAccent).
			Foreground(colorBg).
			Bold(true).
			Padding(0, 1)

	// Normal row
	StyleRow = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	// Labels
	StyleLabel = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true)

	StyleValue = lipgloss.NewStyle().
			Foreground(colorText)

	// Status badges
	StyleBadgeActive = lipgloss.NewStyle().
				Background(colorSuccess).
				Foreground(colorBg).
				Bold(true).
				Padding(0, 1)

	StyleBadgeError = lipgloss.NewStyle().
			Background(colorError).
			Foreground(colorBg).
			Bold(true).
			Padding(0, 1)

	StyleBadgeConnecting = lipgloss.NewStyle().
				Background(colorWarning).
				Foreground(colorBg).
				Bold(true).
				Padding(0, 1)

	StyleBadgeStopped = lipgloss.NewStyle().
				Background(colorMuted).
				Foreground(colorBg).
				Bold(true).
				Padding(0, 1)

	// Checked / unchecked
	StyleChecked   = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	StyleUnchecked = lipgloss.NewStyle().Foreground(colorMuted)

	// Keybind help bar
	StyleHelp = lipgloss.NewStyle().
			Foreground(colorMuted).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			Margin(1, 0, 0, 0)

	// Form field active / inactive
	StyleFieldActive = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1)

	StyleFieldInactive = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Padding(0, 1)

	// Title
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent2).
			Margin(0, 0, 1, 0)

	// Subtitle
	StyleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	// Divider
	StyleDivider = lipgloss.NewStyle().
			Foreground(colorBorder)

	// Accent
	StyleAccent = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	StyleMuted  = lipgloss.NewStyle().Foreground(colorMuted)

	StyleSuccess = lipgloss.NewStyle().Foreground(colorSuccess)
	StyleError   = lipgloss.NewStyle().Foreground(colorError)
)
