package ui

import "github.com/charmbracelet/lipgloss"

// Color palette
const (
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSuccess   = lipgloss.Color("#04B575")
	ColorError     = lipgloss.Color("#FF4757")
	ColorWarning   = lipgloss.Color("#FFA502")
	ColorInfo      = lipgloss.Color("#5352ED")
	ColorMuted     = lipgloss.Color("#95A5A6")
	ColorHighlight = lipgloss.Color("#FAFAFA")
	ColorDark      = lipgloss.Color("#2C3E50")
)

// Text styles
var (
	// TitleStyle is used for main titles and headers
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// SubtitleStyle is used for section headers
	SubtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorInfo).
			MarginTop(1).
			MarginBottom(1)

	// SuccessStyle is used for success messages
	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess)

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError)

	// WarningStyle is used for warning messages
	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarning)

	// InfoStyle is used for informational messages
	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

	// MutedStyle is used for secondary/muted text
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Faint(true)

	// BoldStyle is used for emphasis
	BoldStyle = lipgloss.NewStyle().
			Bold(true)
)

// Container styles
var (
	// BoxStyle is used for content boxes with rounded borders
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	// SuccessBoxStyle is used for success message boxes
	SuccessBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	// ErrorBoxStyle is used for error message boxes
	ErrorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	// WarningBoxStyle is used for warning message boxes
	WarningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)
)

// Table styles
var (
	// TableHeaderStyle is used for table headers
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorHighlight).
				Background(ColorPrimary).
				Padding(0, 1)

	// TableCellStyle is used for table cells
	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// TableRowStyle is used for alternating table rows
	TableRowStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight)

	// TableRowAltStyle is used for alternating table rows
	TableRowAltStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)
)

// Badge styles
var (
	// ActiveBadgeStyle is used for "active" status badges
	ActiveBadgeStyle = lipgloss.NewStyle().
				Foreground(ColorDark).
				Background(ColorSuccess).
				Padding(0, 1).
				Bold(true)

	// PendingBadgeStyle is used for "pending" status badges
	PendingBadgeStyle = lipgloss.NewStyle().
				Foreground(ColorDark).
				Background(ColorWarning).
				Padding(0, 1).
				Bold(true)

	// ErrorBadgeStyle is used for "error" status badges
	ErrorBadgeStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Background(ColorError).
			Padding(0, 1).
			Bold(true)

	// InfoBadgeStyle is used for "info" status badges
	InfoBadgeStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Background(ColorInfo).
			Padding(0, 1).
			Bold(true)
)

// Icon constants
const (
	IconSuccess  = "✓"
	IconError    = "✗"
	IconWarning  = "⚠"
	IconInfo     = "ℹ"
	IconCheck    = "✓"
	IconCross    = "✗"
	IconArrow    = "→"
	IconBullet   = "•"
	IconRocket   = "🚀"
	IconPackage  = "📦"
	IconConfig   = "⚙"
	IconDownload = "⬇"
	IconUpload   = "⬆"
	IconFolder   = "📁"
	IconFile     = "📄"
	IconLock     = "🔒"
	IconKey      = "🔑"
	IconCloud    = "☁"
	IconServer   = "🖥"
	IconGlobe    = "🌐"
)
