module github.com/instill-ai/model-backend

go 1.23.7

require (
	cloud.google.com/go/longrunning v0.5.12
	github.com/alicebob/miniredis/v2 v2.33.0
	github.com/frankban/quicktest v1.14.6
	github.com/gabriel-vasile/mimetype v1.4.5
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/gogo/status v1.1.1
	github.com/gojuno/minimock/v3 v3.4.0
	github.com/golang-migrate/migrate/v4 v4.17.1
	github.com/golang/mock v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0
	github.com/iancoleman/strcase v0.3.0
	github.com/influxdata/influxdb-client-go/v2 v2.14.0
	github.com/instill-ai/protogen-go v0.3.3-alpha.0.20241211175103-4f1558f81c9c
	github.com/instill-ai/usage-client v0.3.0-alpha
	github.com/instill-ai/x v0.7.0-alpha
	github.com/jackc/pgx/v5 v5.6.0
	github.com/knadh/koanf v1.5.0
	github.com/lestrrat-go/jspointer v0.0.0-20181205001929-82fadba7561c
	github.com/lestrrat-go/jsref v0.0.0-20211028120858-c0bcbb5abf20
	github.com/lestrrat-go/option v1.0.1
	github.com/lestrrat-go/pdebug v0.0.0-20210111095411-35b07dbf089b
	github.com/lestrrat-go/structinfo v0.0.0-20210312050401-7f8bd69d6acb
	github.com/mennanov/fieldmask-utils v1.1.2
	github.com/openfga/api/proto v0.0.0-20240807201305-c96ec773cae9
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.6.1
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/stretchr/testify v1.10.0
	go.einride.tech/aip v0.68.0
	go.opentelemetry.io/contrib/propagators/b3 v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.28.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.28.0
	go.opentelemetry.io/otel/sdk v1.28.0
	go.opentelemetry.io/otel/sdk/metric v1.28.0
	go.temporal.io/api v1.44.1
	go.temporal.io/sdk v1.32.1
	go.temporal.io/sdk/contrib/opentelemetry v0.6.0
	go.uber.org/zap v1.27.0
	golang.org/x/image v0.19.0
	golang.org/x/net v0.36.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240827150818-7e3bb234dfed
	google.golang.org/grpc v1.66.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/guregu/null.v4 v4.0.0
	gorm.io/datatypes v1.2.1
	gorm.io/driver/postgres v1.5.9
	gorm.io/gorm v1.25.11
	gorm.io/plugin/dbresolver v1.5.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/alicebob/gopher-json v0.0.0-20230218143504-906a9b012302 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/influxdata/line-protocol v0.0.0-20200327222509-2487e7298839 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/minio/crc64nvme v1.0.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.87 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/nexus-rpc/sdk-go v0.1.0 // indirect
	github.com/oapi-codegen/runtime v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	golang.org/x/exp v0.0.0-20240808152545-0cdaa3abc0fa // indirect
	golang.org/x/sync v0.11.0 // indirect
)

require (
	github.com/catalinc/hashcash v1.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/docker v26.1.5+incompatible // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0
	golang.org/x/time v0.6.0 // indirect
	google.golang.org/genproto v0.0.0-20240808171019-573a1156607a // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240827150818-7e3bb234dfed
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.5.7 // indirect
)
