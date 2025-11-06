package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"` // 元数据库配置
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DatabaseConfig 元数据库配置
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// Load 加载配置文件（兼容旧接口）
func Load(configPath string) (*Config, error) {
	cfg, _, err := LoadWithFlags(configPath)
	return cfg, err
}

// LoadWithFlags 加载配置文件并返回命令行参数
func LoadWithFlags(configPath string) (*Config, *Flags, error) {
	// 解析命令行参数
	flags := parseFlags()

	// 如果指定了配置文件路径，使用命令行参数
	if flags.ConfigPath != "" {
		configPath = flags.ConfigPath
	}

	var config Config

	// 如果配置文件存在，加载它
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// 设置默认值
	setDefaults(&config)

	// 命令行参数覆盖配置文件
	applyFlags(&config, flags)

	return &config, flags, nil
}

// Flags 命令行参数
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

// parseFlags 解析命令行参数
func parseFlags() *Flags {
	flags := &Flags{}

	// 先读取环境变量
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

	// 定义命令行参数（会覆盖环境变量）
	flag.StringVar(&flags.ConfigPath, "config", flags.ConfigPath, "配置文件路径 (默认: configs/config.yaml)")
	flag.StringVar(&flags.ConfigPath, "c", flags.ConfigPath, "配置文件路径 (简写)")

	flag.StringVar(&flags.Host, "host", flags.Host, "服务器监听地址 (覆盖配置文件)")
	flag.IntVar(&flags.Port, "port", flags.Port, "服务器端口 (覆盖配置文件)")

	flag.StringVar(&flags.LogLevel, "log-level", flags.LogLevel, "日志级别: debug, info, warn, error (覆盖配置文件)")
	flag.StringVar(&flags.LogFormat, "log-format", flags.LogFormat, "日志格式: json, text (覆盖配置文件)")
	flag.StringVar(&flags.LogOutput, "log-output", flags.LogOutput, "日志输出: stdout, stderr, 文件路径 (覆盖配置文件)")

	flag.StringVar(&flags.DBHost, "db-host", "", "元数据库主机 (覆盖配置文件)")
	flag.IntVar(&flags.DBPort, "db-port", 0, "元数据库端口 (覆盖配置文件)")
	flag.StringVar(&flags.DBUser, "db-user", "", "元数据库用户 (覆盖配置文件)")
	flag.StringVar(&flags.DBPassword, "db-password", "", "元数据库密码 (覆盖配置文件)")
	flag.StringVar(&flags.DBName, "db-name", "", "元数据库名称 (覆盖配置文件)")
	flag.StringVar(&flags.DBSSLMode, "db-sslmode", "", "元数据库 SSL 模式 (覆盖配置文件)")

	flag.BoolVar(&flags.ShowVersion, "version", false, "显示版本信息")
	flag.BoolVar(&flags.ShowVersion, "v", false, "显示版本信息 (简写)")

	flag.Parse()

	return flags
}

// setDefaults 设置默认值
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
		config.Database.DBName = "dts_meta"
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

// applyFlags 应用命令行参数（覆盖配置文件）
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

// PrintUsage 打印使用说明
func PrintUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
	fmt.Fprintf(os.Stderr, "  DTS_CONFIG      - 配置文件路径\n")
	fmt.Fprintf(os.Stderr, "  DTS_HOST        - 服务器监听地址\n")
	fmt.Fprintf(os.Stderr, "  DTS_PORT        - 服务器端口\n")
	fmt.Fprintf(os.Stderr, "  DTS_LOG_LEVEL   - 日志级别\n")
	fmt.Fprintf(os.Stderr, "  DTS_LOG_FORMAT  - 日志格式\n")
	fmt.Fprintf(os.Stderr, "  DTS_LOG_OUTPUT  - 日志输出\n")
}

// DSN 返回数据库连接字符串
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}
