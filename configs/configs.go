package configs

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/jinzhu/configor"
)

// ServerConfig defines HTTP server configurations
type ServerConfig struct {
	Port  int
	HTTPS struct {
		Cert string
		Key  string
	}
	CORSOrigins []string
}

// Configs related to database
type DatabaseConfig struct {
	Username string
	Password string
	Host     string
	Port     int
	Name     string
	Version  uint
	TimeZone string
	Pool     struct {
		IdleConnections int
		MaxConnections  int
		ConnLifeTime    time.Duration
	}
}

type TritonServerConfig struct {
	GrpcUri    string
	ModelStore string
}

// AppConfig defines
type AppConfig struct {
	Server       ServerConfig
	Database     DatabaseConfig
	TritonServer TritonServerConfig
}

// Config - Global variable to export
var Config AppConfig

// Init - Assign global config to decoded config struct
func Init() error {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fileRelativePath := fs.String("file", "configs/config.yaml", "configuration file")
	flag.Parse()
	// Remove special `CONFIGOR` prefix for extracting variables from os environment purpose
	os.Setenv("CONFIGOR_ENV_PREFIX", "CFG")
	// For some reason, you can pass env to switch to different configurations
	env, _ := os.LookupEnv("ENV")
	// Load from default path, or you can use `-file=xxx` to another file
	if err := configor.New(&configor.Config{Environment: env}).Load(&Config, *fileRelativePath); err != nil {
		log.Fatal(err)
	}

	return ValidateConfig(&Config)
}

// ValidateConfig is for custom validation rules for the configuration
func ValidateConfig(cfg *AppConfig) error {
	return nil
}
