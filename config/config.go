package config

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/instill-ai/x/minio"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/redis/go-redis/v9"
)

// ServerConfig defines HTTP server configurations
type ServerConfig struct {
	PrivatePort int `koanf:"privateport"`
	PublicPort  int `koanf:"publicport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
	Edition string `koanf:"edition"`
	Usage   struct {
		UsageIdentifierUID string `koanf:"usageidentifieruid"`
		Enabled            bool   `koanf:"enabled"`
		TLSEnabled         bool   `koanf:"tlsenabled"`
		Host               string `koanf:"host"`
		Port               int    `koanf:"port"`
	}
	Debug       bool `koanf:"debug"`
	MaxDataSize int  `koanf:"maxdatasize"`
	Workflow    struct {
		MaxWorkflowTimeout int32 `koanf:"maxworkflowtimeout"`
		MaxWorkflowRetry   int32 `koanf:"maxworkflowretry"`
		MaxActivityRetry   int32 `koanf:"maxactivityretry"`
	}
	InstillCoreHost   string `koanf:"instillcorehost"`
	TaskSchemaVersion string `koanf:"taskschemaversion"`
}

// DatabaseConfig related to database
type DatabaseConfig struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Replica  struct {
		Username             string `koanf:"username"`
		Password             string `koanf:"password"`
		Host                 string `koanf:"host"`
		Port                 int    `koanf:"port"`
		ReplicationTimeFrame int    `koanf:"replicationtimeframe"` // in seconds
	} `koanf:"replica"`
	Name     string `koanf:"name"`
	Version  uint   `koanf:"version"`
	TimeZone string `koanf:"timezone"`
	Pool     struct {
		IdleConnections int           `koanf:"idleconnections"`
		MaxConnections  int           `koanf:"maxconnections"`
		ConnLifeTime    time.Duration `koanf:"connlifetime"`
	}
}

// RayConfig related to Ray server
type RayConfig struct {
	Host string `koanf:"host"`
	Port struct {
		DASHBOARD int `koanf:"dashboard"`
		SERVE     int `koanf:"serve"`
		GRPC      int `koanf:"grpc"`
		CLIENT    int `koanf:"client"`
		GCS       int `koanf:"gcs"`
		METRICS   int `koanf:"metrics"`
	} `koanf:"port"`
	Vram string `koanf:"vram"`
}

// MgmtBackendConfig related to mgmt-backend
type MgmtBackendConfig struct {
	Host        string `koanf:"host"`
	PrivatePort int    `koanf:"privateport"`
	PublicPort  int    `koanf:"publicport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// ArtifactBackendConfig related to artifact-backend
type ArtifactBackendConfig struct {
	Host        string `koanf:"host"`
	PrivatePort int    `koanf:"privateport"`
	PublicPort  int    `koanf:"publicport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// CacheConfig related to Redis
type CacheConfig struct {
	Redis struct {
		RedisOptions redis.Options `koanf:"redisoptions"`
	}
	Model struct {
		Enabled  bool   `koanf:"enabled"`
		CacheDir string `koanf:"cache_dir"`
	}
}

type InitModelConfig struct {
	Path    string `koanf:"path"`
	Enabled bool   `koanf:"enabled"`
}

// TemporalConfig related to Temporal
type TemporalConfig struct {
	HostPort   string `koanf:"hostport"`
	Namespace  string `koanf:"namespace"`
	Retention  string `koanf:"retention"`
	Ca         string `koanf:"ca"`
	Cert       string `koanf:"cert"`
	Key        string `koanf:"key"`
	ServerName string `koanf:"servername"`
}

// LogConfig related to logging
type LogConfig struct {
	External      bool `koanf:"external"`
	OtelCollector struct {
		Host string `koanf:"host"`
		Port string `koanf:"port"`
	}
}

// OpenFGA config
type OpenFGAConfig struct {
	Host    string `koanf:"host"`
	Port    int    `koanf:"port"`
	Replica struct {
		Host                 string `koanf:"host"`
		Port                 int    `koanf:"port"`
		ReplicationTimeFrame int    `koanf:"replicationtimeframe"` // in seconds
	} `koanf:"replica"`
}

// Registry config
type RegistryConfig struct {
	Host string `koanf:"host"`
	Port int    `koanf:"port"`
}

// InfluxDBConfig defines the InfluxDB configuration.
type InfluxDBConfig struct {
	URL           string        `koanf:"url"`
	Token         string        `koanf:"token"`
	Org           string        `koanf:"org"`
	Bucket        string        `koanf:"bucket"`
	FlushInterval time.Duration `koanf:"flushinterval"`
	HTTPS         struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// AppConfig defines
type AppConfig struct {
	Server          ServerConfig          `koanf:"server"`
	Database        DatabaseConfig        `koanf:"database"`
	Ray             RayConfig             `koanf:"ray"`
	MgmtBackend     MgmtBackendConfig     `koanf:"mgmtbackend"`
	ArtifactBackend ArtifactBackendConfig `koanf:"artifactbackend"`
	Cache           CacheConfig           `koanf:"cache"`
	Temporal        TemporalConfig        `koanf:"temporal"`
	OpenFGA         OpenFGAConfig         `koanf:"openfga"`
	Registry        RegistryConfig        `koanf:"registry"`
	InitModel       InitModelConfig       `koanf:"initmodel"`
	Log             LogConfig             `koanf:"log"`
	Minio           minio.Config          `koanf:"minio"`
	InfluxDB        InfluxDBConfig        `koanf:"influxdb"`
}

// Config - Global variable to export
var Config AppConfig

// Init - Assign global config to decoded config struct
func Init(filePath string) error {
	k := koanf.New(".")
	parser := yaml.Parser()

	if err := k.Load(confmap.Provider(map[string]any{
		"database.replica.replicationtimeframe": 60,
		"openfga.replica.replicationtimeframe":  60,
	}, "."), nil); err != nil {
		log.Fatal(err.Error())
	}

	if err := k.Load(file.Provider(filePath), parser); err != nil {
		log.Fatal(err.Error())
	}

	if err := k.Load(env.ProviderWithValue("CFG_", ".", func(s string, v string) (string, any) {
		key := strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "CFG_")), "_", ".")
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
// for future use
func ValidateConfig(_ *AppConfig) error {
	return nil
}

var defaultConfigPath = "config/config.yaml"

// ParseConfigFlag allows clients to specify the relative path to the file from
// which the configuration will be loaded.
func ParseConfigFlag() string {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	configPath := fs.String("file", defaultConfigPath, "configuration file")
	flag.Parse()

	return *configPath
}
