module github.com/instill-ai/model-backend

go 1.22.5

require (
	cloud.google.com/go/longrunning v0.5.4
	github.com/alicebob/miniredis/v2 v2.33.0
	github.com/gabriel-vasile/mimetype v1.4.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/gogo/status v1.1.1
	github.com/golang-migrate/migrate/v4 v4.15.2
	github.com/golang/mock v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1
	github.com/iancoleman/strcase v0.3.0
	github.com/instill-ai/protogen-go v0.3.3-alpha.0.20240724130416-27b3d8e33885
	github.com/instill-ai/usage-client v0.2.4-alpha.0.20240123081026-6c78d9a5197a
	github.com/instill-ai/x v0.4.0-alpha
	github.com/jackc/pgx/v5 v5.6.0
	github.com/knadh/koanf v1.5.0
	github.com/mennanov/fieldmask-utils v1.1.0
	github.com/openfga/api/proto v0.0.0-20240501220219-2b164f5813a7
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.5.1
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/stretchr/testify v1.8.4
	go.einride.tech/aip v0.67.1
	go.opentelemetry.io/contrib/propagators/b3 v1.17.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.24.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/sdk/metric v1.24.0
	go.temporal.io/api v1.18.2-0.20230324225508-f2c7ab685b44
	go.temporal.io/sdk v1.21.1
	go.temporal.io/sdk/contrib/opentelemetry v0.2.0
	go.uber.org/zap v1.26.0
	golang.org/x/image v0.18.0
	golang.org/x/net v0.23.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240125205218-1f4bbc51befe
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
	gorm.io/datatypes v1.1.0
	gorm.io/driver/postgres v1.4.5
	gorm.io/gorm v1.25.2
	gorm.io/plugin/dbresolver v1.5.1
)

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.4 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
)

require (
	github.com/catalinc/hashcash v0.0.0-20220723060415-5e3ec3e24f67 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/docker v25.0.5+incompatible // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.14.3 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgtype v1.14.0 // indirect
	github.com/jackc/pgx/v4 v4.18.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/lib/pq v1.10.7 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0
	go.opentelemetry.io/proto/otlp v1.1.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240125205218-1f4bbc51befe
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.4.4 // indirect
)
