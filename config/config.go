package config

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

// ServerConfig defines HTTP server configurations
type ServerConfig struct {
	Port  int `koanf:"port"`
	HTTPS struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
	CORSOrigins  []string `koanf:"corsorigins"`
	Edition      string   `koanf:"edition"`
	DisableUsage bool     `koanf:"disableusage"`
	Debug        bool     `koanf:"debug"`
}

// config related to database
type DatabaseConfig struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Name     string `koanf:"name"`
	Version  uint   `koanf:"version"`
	TimeZone string `koanf:"timezone"`
	Pool     struct {
		IdleConnections int           `koanf:"idleconnections"`
		MaxConnections  int           `koanf:"maxconnections"`
		ConnLifeTime    time.Duration `koanf:"connlifetime"`
	}
}

type TritonServerConfig struct {
	GrpcUri    string `koanf:"grpcuri"`
	ModelStore string `koanf:"modelstore"`
}

// MgmtBackendConfig related to mgmt-backend
type MgmtBackendConfig struct {
	Host  string `koanf:"host"`
	Port  int    `koanf:"port"`
	HTTPS struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// CacheConfig related to Redis
type CacheConfig struct {
	Redis struct {
		RedisOptions redis.Options `koanf:"redisoptions"`
	}
}

type UsageBackendConfig struct {
	TLSEnabled bool   `koanf:"tlsenabled"`
	Host       string `koanf:"host"`
	Port       int    `koanf:"port"`
}

type PipelineBackendConfig struct {
	Host  string `koanf:"host"`
	Port  int    `koanf:"port"`
	HTTPS struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// AppConfig defines
type AppConfig struct {
	Server          ServerConfig          `koanf:"server"`
	Database        DatabaseConfig        `koanf:"database"`
	TritonServer    TritonServerConfig    `koanf:"tritonserver"`
	MgmtBackend     MgmtBackendConfig     `koanf:"mgmtbackend"`
	Cache           CacheConfig           `koanf:"cache"`
	UsageBackend    UsageBackendConfig    `koanf:"usagebackend"`
	PipelineBackend PipelineBackendConfig `koanf:"pipelinebackend"`
}

// Config - Global variable to export
var Config AppConfig

// Init - Assign global config to decoded config struct
func Init() error {
	k := koanf.New(".")
	parser := yaml.Parser()

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fileRelativePath := fs.String("file", "config/config.yaml", "configuration file")
	flag.Parse()

	if err := k.Load(file.Provider(*fileRelativePath), parser); err != nil {
		log.Fatal(err.Error())
	}

	if err := k.Load(env.ProviderWithValue("CFG_", ".", func(s string, v string) (string, interface{}) {
		key := strings.Replace(strings.ToLower(strings.TrimPrefix(s, "CFG_")), "_", ".", -1)
		if strings.Contains(v, ",") {
			return key, strings.Split(strings.TrimSpace(v), ",")
		}
		return key, v
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
