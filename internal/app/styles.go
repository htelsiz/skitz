package app

import "github.com/charmbracelet/lipgloss"

// View modes
const (
	viewDashboard = iota
	viewDetail
)

// Colors
var (
	primary   = lipgloss.Color("99")  // Purple (Toad-style)
	secondary = lipgloss.Color("114") // Green
	subtle    = lipgloss.Color("242") // Gray
	dimBorder = lipgloss.Color("238") // Subtle borders
	white     = lipgloss.Color("255")
)

// Styles
var (
	// Dashboard sidebar pane
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimBorder)
)

// customStyleJSON is the custom glamour style for premium markdown rendering
var customStyleJSON = `{
	"document": {
		"margin": 0,
		"block_prefix": "",
		"block_suffix": ""
	},
	"heading": {
		"block_suffix": "\n",
		"color": "99",
		"bold": true
	},
	"h1": {
		"prefix": "══════════════════════════════════════════\n  ",
		"suffix": "\n══════════════════════════════════════════",
		"color": "99",
		"bold": true,
		"block_suffix": "\n"
	},
	"h2": {
		"prefix": "▌ ",
		"color": "114",
		"bold": true,
		"block_suffix": "\n"
	},
	"h3": {
		"prefix": "  ◆ ",
		"color": "252",
		"bold": true
	},
	"paragraph": {
		"block_prefix": "",
		"block_suffix": "\n"
	},
	"list": {
		"level_indent": 2
	},
	"item": {
		"block_prefix": "  "
	},
	"enumeration": {
		"block_prefix": "  "
	},
	"code": {
		"color": "212",
		"background_color": "236",
		"prefix": " ",
		"suffix": " "
	},
	"code_block": {
		"color": "252",
		"background_color": "235",
		"margin": 1,
		"chroma": {
			"text": { "color": "#e0e0e0" },
			"keyword": { "color": "#50fa7b", "bold": true },
			"name": { "color": "#5fd7ff" },
			"name_builtin": { "color": "#5fd7ff" },
			"name_class": { "color": "#5fd7ff", "bold": true },
			"name_function": { "color": "#50fa7b" },
			"name_variable": { "color": "#ffff87" },
			"literal_string": { "color": "#ffff87" },
			"literal_number": { "color": "#ffaf5f" },
			"comment": { "color": "#6272a4", "italic": true },
			"operator": { "color": "#ff79c6" },
			"punctuation": { "color": "#f8f8f2" },
			"generic_heading": { "color": "#ff79c6", "bold": true },
			"generic_subheading": { "color": "#5fd7ff", "bold": true },
			"generic_deleted": { "color": "#ff5555" },
			"generic_inserted": { "color": "#50fa7b" },
			"generic_emph": { "italic": true },
			"generic_strong": { "bold": true },
			"name_tag": { "color": "#ff79c6" },
			"name_attribute": { "color": "#50fa7b" },
			"name_constant": { "color": "#ff79c6" },
			"literal_string_escape": { "color": "#ff79c6", "bold": true }
		}
	},
	"table": {
		"color": "252",
		"center_separator": "┼",
		"column_separator": "│",
		"row_separator": "─"
	},
	"link": {
		"color": "39",
		"underline": true
	},
	"link_text": {
		"color": "99",
		"bold": true
	},
	"emph": {
		"color": "252",
		"italic": true
	},
	"strong": {
		"color": "255",
		"bold": true
	},
	"hr": {
		"color": "238",
		"format": "────────────────────────────────────────"
	},
	"block_quote": {
		"color": "114",
		"indent": 2,
		"indent_token": "▎ "
	}
}`
