package configs

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
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
	logger, _ := logger.GetZapLogger()

	k := koanf.New(".")
	parser := yaml.Parser()

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fileRelativePath := fs.String("file", "configs/config.yaml", "configuration file")
	flag.Parse()

	if err := k.Load(file.Provider(*fileRelativePath), parser); err != nil {
		logger.Fatal(err.Error())
	}

	if err := k.Load(env.Provider("CFG_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "CFG_")), "_", ".", -1)
	}), nil); err != nil {
		return err
	}

	if err := k.Unmarshal("", &Config); err != nil {
		return err
	}

	return ValidateConfig(&Config)
}

// ValidateConfig is for custom validation rules for the configuration
func ValidateConfig(cfg *AppConfig) error {
	return nil
}
