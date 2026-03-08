package config

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// Config is the root configuration structure.
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	App    AppConfig    `mapstructure:"app"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type AppConfig struct {
	Backend        string   `mapstructure:"backend"`
	DefaultPrinter string   `mapstructure:"default_printer"`
	FontDirs       []string `mapstructure:"font_dirs"`
}

// getDefaultFontDirs returns OS-appropriate font directories
func getDefaultFontDirs() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			"./fonts",
			"C:\\Windows\\Fonts",
		}
	case "darwin": // macOS
		return []string{
			"./fonts",
			"/Library/Fonts",
			"~/Library/Fonts",
			"/System/Library/Fonts/Supplemental",
		}
	default: // Linux and others
		return []string{
			"./fonts",
			"/usr/share/fonts/truetype",
			"/usr/local/share/fonts",
			"~/.local/share/fonts",
			"~/.fonts",
		}
	}
}

// LoadConfig loads configuration from files and environment variables.
// Returns a new Config — does NOT store in a global variable.
func LoadConfig() (*Config, error) {
	slog.Info("Loading configuration with priority order: default -> config file -> environment variables -> API parameters")

	v := viper.New()

	// Search for config in multiple locations
	v.AddConfigPath("/etc/labelprinter/")
	v.AddConfigPath("$HOME/.labelprinter")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	v.SetConfigName("config") // Looks for config.json, config.yaml etc
	v.SetConfigType("json")

	// Set defaults first (lowest priority)
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "localhost")
	v.SetDefault("app.font_dirs", getDefaultFontDirs())
	v.SetDefault("app.default_printer", "")
	v.SetDefault("app.backend", "auto")

	// Configure environment variables (middle priority)
	v.SetEnvPrefix("LABELPRINTER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to read config file (higher priority than defaults but lower than env vars)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("No config file found, using defaults")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		slog.Info("Using config file", "path", v.ConfigFileUsed())
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	logConfigSources(v, &cfg)
	return &cfg, nil
}

func logConfigSources(v *viper.Viper, cfg *Config) {
	if strings.ToUpper(os.Getenv("LOG_LEVEL")) == "ERROR" {
		return
	}

	slog.Info("Configuration loaded with the following values:")
	logConfigValue(v, "server.port", fmt.Sprintf("%d", cfg.Server.Port))
	logConfigValue(v, "server.host", cfg.Server.Host)
	logConfigValue(v, "app.backend", cfg.App.Backend)
	logConfigValue(v, "app.default_printer", cfg.App.DefaultPrinter)
	slog.Info("Configuration value", "key", "app.font_dirs", "value", cfg.App.FontDirs)
}

func logConfigValue(v *viper.Viper, key string, value string) {
	source := "default"

	if v.InConfig(key) {
		source = fmt.Sprintf("config file (%s)", v.ConfigFileUsed())
	}

	// Check corresponding environment variable
	envKey := "LABELPRINTER_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if _, exists := os.LookupEnv(envKey); exists {
		source = fmt.Sprintf("environment (%s)", envKey)
	}

	slog.Info("Configuration value", "key", key, "value", value, "source", source)
}
