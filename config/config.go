package config

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// Global config instance
var Cfg Config

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type AppConfig struct {
	FontDirs       []string `mapstructure:"font_dirs"`
	DefaultPrinter string   `mapstructure:"default_printer"`
	Backend        string   `mapstructure:"backend"`
}

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	App    AppConfig    `mapstructure:"app"`
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

// LoadConfig reads configuration and stores it in Cfg with priority:
// default -> config file -> environment variables -> API parameters
func LoadConfig() error {
	log.Println("Loading configuration with priority order: default -> config file -> environment variables -> API parameters")
	
	// Search for config in multiple locations
	viper.AddConfigPath("/etc/labelprinter/")
	viper.AddConfigPath("$HOME/.labelprinter")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	
	viper.SetConfigName("config") // Looks for config.json, config.yaml etc
	viper.SetConfigType("json")
	
	// Set defaults first (lowest priority)
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("app.font_dirs", getDefaultFontDirs())
	viper.SetDefault("app.default_printer", "")
	viper.SetDefault("app.backend", "auto")

	// Configure environment variables (middle priority)
	viper.SetEnvPrefix("LABELPRINTER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Try to read config file (higher priority than defaults but lower than env vars)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("No config file found, using defaults")
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	logConfigSources()
	return nil
}

func logConfigSources() {
	if strings.ToUpper(os.Getenv("LOG_LEVEL")) == "ERROR" {
		return
	}

	log.Printf("Configuration loaded with the following values:")
	logConfigValue("server.port", fmt.Sprintf("%d", Cfg.Server.Port))
	logConfigValue("server.host", Cfg.Server.Host)
	logConfigValue("app.backend", Cfg.App.Backend)
	logConfigValue("app.default_printer", Cfg.App.DefaultPrinter)
	log.Printf("  - app.font_dirs: %v", Cfg.App.FontDirs)
}

func logConfigValue(key string, value string) {
	source := "default"
	
	if viper.InConfig(key) {
		source = fmt.Sprintf("config file (%s)", viper.ConfigFileUsed())
	}
	
	// Check corresponding environment variable
	envKey := "LABELPRINTER_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if _, exists := os.LookupEnv(envKey); exists {
		source = fmt.Sprintf("environment (%s)", envKey)
	}
	
	log.Printf("  - %s: %s (from %s)", key, value, source)
}
