package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"` // Metadata database configuration
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DatabaseConfig represents metadata database configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// LogConfig represents log configuration
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// Load loads configuration file (compatible with old interface)
func Load(configPath string) (*Config, error) {
	cfg, _, err := LoadWithFlags(configPath)
	return cfg, err
}

// LoadWithFlags loads configuration file and returns command line flags
func LoadWithFlags(configPath string) (*Config, *Flags, error) {
	// Parse command line arguments
	flags := parseFlags()

	// If config file path is specified, use command line argument
	if flags.ConfigPath != "" {
		configPath = flags.ConfigPath
	}

	var config Config

	// If config file exists, load it
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Set default values
	setDefaults(&config)

	// Command line arguments override config file
	applyFlags(&config, flags)

	return &config, flags, nil
}

// Flags represents command line flags
type Flags struct {
	ConfigPath  string
	Host        string
	Port        int
	LogLevel    string
	LogFormat   string
	LogOutput   string
	DBHost      string
	DBPort      int
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	ShowVersion bool
}

// parseFlags parses command line arguments
func parseFlags() *Flags {
	flags := &Flags{}

	// Read environment variables first
	flags.ConfigPath = os.Getenv("DTS_CONFIG")
	if flags.Host == "" {
		flags.Host = os.Getenv("DTS_HOST")
	}
	if portStr := os.Getenv("DTS_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			flags.Port = port
		}
	}
	flags.LogLevel = os.Getenv("DTS_LOG_LEVEL")
	flags.LogFormat = os.Getenv("DTS_LOG_FORMAT")
	flags.LogOutput = os.Getenv("DTS_LOG_OUTPUT")

	// Define command line arguments (will override environment variables)
	flag.StringVar(&flags.ConfigPath, "config", flags.ConfigPath, "Config file path (default: configs/config.yaml)")
	flag.StringVar(&flags.ConfigPath, "c", flags.ConfigPath, "Config file path (short)")

	flag.StringVar(&flags.Host, "host", flags.Host, "Server listen address (overrides config file)")
	flag.IntVar(&flags.Port, "port", flags.Port, "Server port (overrides config file)")

	flag.StringVar(&flags.LogLevel, "log-level", flags.LogLevel, "Log level: debug, info, warn, error (overrides config file)")
	flag.StringVar(&flags.LogFormat, "log-format", flags.LogFormat, "Log format: json, text (overrides config file)")
	flag.StringVar(&flags.LogOutput, "log-output", flags.LogOutput, "Log output: stdout, stderr, file path (overrides config file)")

	flag.StringVar(&flags.DBHost, "db-host", "", "Metadata database host (overrides config file)")
	flag.IntVar(&flags.DBPort, "db-port", 0, "Metadata database port (overrides config file)")
	flag.StringVar(&flags.DBUser, "db-user", "", "Metadata database user (overrides config file)")
	flag.StringVar(&flags.DBPassword, "db-password", "", "Metadata database password (overrides config file)")
	flag.StringVar(&flags.DBName, "db-name", "", "Metadata database name (overrides config file)")
	flag.StringVar(&flags.DBSSLMode, "db-sslmode", "", "Metadata database SSL mode (overrides config file)")

	flag.BoolVar(&flags.ShowVersion, "version", false, "Show version information")
	flag.BoolVar(&flags.ShowVersion, "v", false, "Show version information (short)")

	flag.Parse()

	return flags
}

// setDefaults sets default values
func setDefaults(config *Config) {
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Database.Host == "" {
		config.Database.Host = "localhost"
	}
	if config.Database.Port == 0 {
		config.Database.Port = 5432
	}
	if config.Database.User == "" {
		config.Database.User = "postgres"
	}
	if config.Database.Password == "" {
		config.Database.Password = "postgres"
	}
	if config.Database.DBName == "" {
		config.Database.DBName = "postgres"
	}
	if config.Database.SSLMode == "" {
		config.Database.SSLMode = "disable"
	}
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}
	if config.Log.Format == "" {
		config.Log.Format = "json"
	}
	if config.Log.Output == "" {
		config.Log.Output = "stdout"
	}
}

// applyFlags applies command line arguments (overrides config file)
func applyFlags(config *Config, flags *Flags) {
	if flags.Host != "" {
		config.Server.Host = flags.Host
	}
	if flags.Port > 0 {
		config.Server.Port = flags.Port
	}
	if flags.LogLevel != "" {
		config.Log.Level = strings.ToLower(flags.LogLevel)
	}
	if flags.LogFormat != "" {
		config.Log.Format = strings.ToLower(flags.LogFormat)
	}
	if flags.LogOutput != "" {
		config.Log.Output = flags.LogOutput
	}
	if flags.DBHost != "" {
		config.Database.Host = flags.DBHost
	}
	if flags.DBPort > 0 {
		config.Database.Port = flags.DBPort
	}
	if flags.DBUser != "" {
		config.Database.User = flags.DBUser
	}
	if flags.DBPassword != "" {
		config.Database.Password = flags.DBPassword
	}
	if flags.DBName != "" {
		config.Database.DBName = flags.DBName
	}
	if flags.DBSSLMode != "" {
		config.Database.SSLMode = flags.DBSSLMode
	}
}

// PrintUsage prints usage information
func PrintUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
	fmt.Fprintf(os.Stderr, "  DTS_CONFIG      - Config file path\n")
	fmt.Fprintf(os.Stderr, "  DTS_HOST        - Server listen address\n")
	fmt.Fprintf(os.Stderr, "  DTS_PORT        - Server port\n")
	fmt.Fprintf(os.Stderr, "  DTS_LOG_LEVEL   - Log level\n")
	fmt.Fprintf(os.Stderr, "  DTS_LOG_FORMAT  - Log format\n")
	fmt.Fprintf(os.Stderr, "  DTS_LOG_OUTPUT  - Log output\n")
}

// DSN returns database connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}
