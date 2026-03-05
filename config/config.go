package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/key"
)

// KeyBinding represents a single configurable key binding.
type KeyBinding struct {
	Keys []string `toml:"keys"`
	Help string   `toml:"help"`
}

// KeyBindings holds all configurable keybindings for the application.
type KeyBindings struct {
	Up       KeyBinding `toml:"up"`
	Down     KeyBinding `toml:"down"`
	NextPane KeyBinding `toml:"next_pane"`
	PrevPane KeyBinding `toml:"prev_pane"`
	Select   KeyBinding `toml:"select"`
	Scan     KeyBinding `toml:"scan"`
	Quit     KeyBinding `toml:"quit"`
}

// Colors holds all color configurations for the application.
type Colors struct {
	Primary     string `toml:"primary"`      // Default text and UI elements
	Active      string `toml:"active"`       // Active/selected borders
	ActiveText  string `toml:"active_text"`  // Active/selected text
	SelectionBg string `toml:"selection_bg"` // Selection bar background
	Inactive    string `toml:"inactive"`     // Inactive/dimmed elements
	Error       string `toml:"error"`        // Error border color
	ErrorText   string `toml:"error_text"`   // Error message text color
	HelpText    string `toml:"help_text"`    // Help text at bottom of window
}

// Config holds the entire application configuration.
type Config struct {
	KeyBindings KeyBindings `toml:"keybindings"`
	Colors      Colors      `toml:"colors"`
}

// DefaultKeyBindings returns the default keybinding configuration.
func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		Up:       KeyBinding{Keys: []string{"k", "up"}, Help: "Up"},
		Down:     KeyBinding{Keys: []string{"j", "down"}, Help: "Down"},
		NextPane: KeyBinding{Keys: []string{"tab"}, Help: "Next"},
		PrevPane: KeyBinding{Keys: []string{"shift+tab"}, Help: "Prev"},
		Select:   KeyBinding{Keys: []string{"enter", " "}, Help: "Dis/Connect"},
		Scan:     KeyBinding{Keys: []string{"s"}, Help: "Scan"},
		Quit:     KeyBinding{Keys: []string{"q", "ctrl+c", "ctrl+q", "ctrl+w"}, Help: "Quit"},
	}
}

// DefaultColors returns the default color configuration.
func DefaultColors() Colors {
	return Colors{
		Primary:     "#a7abca",
		Active:      "#9cca69",
		ActiveText:  "#cda162",
		SelectionBg: "#5a6988",
		Inactive:    "#444a66",
		Error:       "#ff0000",
		ErrorText:   "#aa0000",
		HelpText:    "#a7abca",
	}
}

// DefaultConfig returns a new Config with default values.
func DefaultConfig() Config {
	return Config{
		KeyBindings: DefaultKeyBindings(),
		Colors:      DefaultColors(),
	}
}

// GetConfigPath returns the path to the config file following XDG specification.
func GetConfigPath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "bluepala", "config.toml"), nil
}

// Load loads the configuration from the config file.
// If the file doesn't exist, it creates one with default values.
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		cfg := DefaultConfig()
		return &cfg, nil
	}

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		cfg := DefaultConfig()
		if saveErr := Save(&cfg); saveErr != nil {
			return &cfg, nil
		}
		return &cfg, nil
	}

	var cfg Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		cfg = DefaultConfig()
		return &cfg, nil
	}

	cfg = mergeWithDefaults(cfg)
	return &cfg, nil
}

// Save saves the configuration to the config file.
func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	header := `# Bluepala Configuration File
# Keybindings and colors can be customized below.
# Available modifiers: ctrl, alt, shift
# Examples: "ctrl+c", "shift+tab", "a", "up", "down"
# Special keys: enter, space (use " "), tab, backspace, esc
# Multiple keys can be assigned to the same action.

`
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	encoder := toml.NewEncoder(file)
	encoder.Indent = "  "
	return encoder.Encode(cfg)
}

// mergeWithDefaults ensures all fields have values, using defaults for missing ones.
func mergeWithDefaults(cfg Config) Config {
	defaults := DefaultKeyBindings()
	defaultColors := DefaultColors()

	if len(cfg.KeyBindings.Up.Keys) == 0 {
		cfg.KeyBindings.Up = defaults.Up
	}
	if len(cfg.KeyBindings.Down.Keys) == 0 {
		cfg.KeyBindings.Down = defaults.Down
	}
	if len(cfg.KeyBindings.NextPane.Keys) == 0 {
		cfg.KeyBindings.NextPane = defaults.NextPane
	}
	if len(cfg.KeyBindings.PrevPane.Keys) == 0 {
		cfg.KeyBindings.PrevPane = defaults.PrevPane
	}
	if len(cfg.KeyBindings.Select.Keys) == 0 {
		cfg.KeyBindings.Select = defaults.Select
	}
	if len(cfg.KeyBindings.Scan.Keys) == 0 {
		cfg.KeyBindings.Scan = defaults.Scan
	}
	if len(cfg.KeyBindings.Quit.Keys) == 0 {
		cfg.KeyBindings.Quit = defaults.Quit
	}

	// Merge help text
	if cfg.KeyBindings.Up.Help == "" {
		cfg.KeyBindings.Up.Help = defaults.Up.Help
	}
	if cfg.KeyBindings.Down.Help == "" {
		cfg.KeyBindings.Down.Help = defaults.Down.Help
	}
	if cfg.KeyBindings.NextPane.Help == "" {
		cfg.KeyBindings.NextPane.Help = defaults.NextPane.Help
	}
	if cfg.KeyBindings.PrevPane.Help == "" {
		cfg.KeyBindings.PrevPane.Help = defaults.PrevPane.Help
	}
	if cfg.KeyBindings.Select.Help == "" {
		cfg.KeyBindings.Select.Help = defaults.Select.Help
	}
	if cfg.KeyBindings.Scan.Help == "" {
		cfg.KeyBindings.Scan.Help = defaults.Scan.Help
	}
	if cfg.KeyBindings.Quit.Help == "" {
		cfg.KeyBindings.Quit.Help = defaults.Quit.Help
	}

	// Merge colors
	if cfg.Colors.Primary == "" {
		cfg.Colors.Primary = defaultColors.Primary
	}
	if cfg.Colors.Active == "" {
		cfg.Colors.Active = defaultColors.Active
	}
	if cfg.Colors.ActiveText == "" {
		cfg.Colors.ActiveText = defaultColors.ActiveText
	}
	if cfg.Colors.SelectionBg == "" {
		cfg.Colors.SelectionBg = defaultColors.SelectionBg
	}
	if cfg.Colors.Inactive == "" {
		cfg.Colors.Inactive = defaultColors.Inactive
	}
	if cfg.Colors.Error == "" {
		cfg.Colors.Error = defaultColors.Error
	}
	if cfg.Colors.ErrorText == "" {
		cfg.Colors.ErrorText = defaultColors.ErrorText
	}
	if cfg.Colors.HelpText == "" {
		cfg.Colors.HelpText = defaultColors.HelpText
	}

	return cfg
}

// ToKeyBinding converts a KeyBinding to a bubbles key.Binding.
func (kb KeyBinding) ToKeyBinding() key.Binding {
	return key.NewBinding(
		key.WithKeys(kb.Keys...),
		key.WithHelp(formatKeysHelp(kb.Keys), kb.Help),
	)
}

// formatKeysHelp creates a compact display string from a key list.
func formatKeysHelp(keys []string) string {
	if len(keys) == 0 {
		return ""
	}

	symbolMap := map[string]string{
		"up":        "↑",
		"down":      "↓",
		"left":      "←",
		"right":     "→",
		"enter":     "⤶",
		" ":         "␣",
		"space":     "␣",
		"tab":       "⇥",
		"shift+tab": "⇤",
		"backspace": "⌫",
		"delete":    "⌦",
		"esc":       "⎋",
		"ctrl+c":    "^C",
		"ctrl+q":    "^Q",
		"ctrl+w":    "^W",
	}

	result := ""
	shown := 0
	for _, k := range keys {
		if shown >= 2 {
			break
		}
		if shown > 0 {
			result += "/"
		}
		if sym, ok := symbolMap[k]; ok {
			result += sym
		} else {
			result += k
		}
		shown++
	}
	return result
}

// Matches checks if a raw key string matches this binding.
func (kb KeyBinding) Matches(keyStr string) bool {
	for _, k := range kb.Keys {
		if k == keyStr {
			return true
		}
	}
	return false
}

// AppKeyMap holds the application's key bindings in bubbles format.
type AppKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	NextPane key.Binding
	PrevPane key.Binding
	Select   key.Binding
	Scan     key.Binding
	Quit     key.Binding
}

// NewAppKeyMap creates a new AppKeyMap from a Config.
func NewAppKeyMap(cfg *Config) AppKeyMap {
	return AppKeyMap{
		Up:       cfg.KeyBindings.Up.ToKeyBinding(),
		Down:     cfg.KeyBindings.Down.ToKeyBinding(),
		NextPane: cfg.KeyBindings.NextPane.ToKeyBinding(),
		PrevPane: cfg.KeyBindings.PrevPane.ToKeyBinding(),
		Select:   cfg.KeyBindings.Select.ToKeyBinding(),
		Scan:     cfg.KeyBindings.Scan.ToKeyBinding(),
		Quit:     cfg.KeyBindings.Quit.ToKeyBinding(),
	}
}

// ShortHelp returns key bindings to show in the status bar.
func (k AppKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Scan, k.NextPane, k.Quit}
}

// FullHelp returns the full set of key bindings.
func (k AppKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.NextPane, k.PrevPane, k.Select, k.Scan, k.Quit},
	}
}
