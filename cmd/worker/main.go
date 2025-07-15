package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
	"gorm.io/gorm"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"

	database "github.com/instill-ai/model-backend/pkg/db"
	modelWorker "github.com/instill-ai/model-backend/pkg/worker"
	logx "github.com/instill-ai/x/log"
	miniox "github.com/instill-ai/x/minio"
	otelx "github.com/instill-ai/x/otel"
	temporalx "github.com/instill-ai/x/temporal"
)

const gracefulShutdownWaitPeriod = 15 * time.Second
const gracefulShutdownTimeout = 60 * time.Minute

var (
	// These variables might be overridden at buildtime.
	serviceName    = "model-backend-worker"
	serviceVersion = "dev"
)

func main() {
	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup all OpenTelemetry components
	cleanup := otelx.SetupWithCleanup(ctx,
		otelx.WithServiceName(serviceName),
		otelx.WithServiceVersion(serviceVersion),
		otelx.WithHost(config.Config.OTELCollector.Host),
		otelx.WithPort(config.Config.OTELCollector.Port),
		otelx.WithCollectorEnable(config.Config.OTELCollector.Enable),
	)
	defer cleanup()

	logx.Debug = config.Config.Server.Debug
	logger, _ := logx.GetZapLogger(ctx)
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	// Set gRPC logging based on debug mode
	if config.Config.Server.Debug {
		grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 0) // All logs
	} else {
		grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 3) // verbosity 3 will avoid [transport] from emitting
	}

	datamodel.InitJSONSchema(ctx)

	// Initialize all clients
	redisClient, db, rayService, temporalClient, minioClient, influxDB, closeClients := newClients(ctx, logger)
	defer closeClients()

	// for only local temporal cluster
	if config.Config.Temporal.ServerRootCA == "" && config.Config.Temporal.ClientCert == "" && config.Config.Temporal.ClientKey == "" {
		initTemporalNamespace(ctx, temporalClient, logger)
	}

	repo := repository.NewRepository(db, redisClient)
	cw := modelWorker.NewWorker(redisClient, rayService, repo, influxDB.WriteAPI(), minioClient, nil)

	w := worker.New(temporalClient, modelWorker.TaskQueue, worker.Options{
		WorkerStopTimeout:                      gracefulShutdownTimeout,
		MaxConcurrentWorkflowTaskExecutionSize: 100,
		Interceptors: func() []interceptor.WorkerInterceptor {
			if !config.Config.OTELCollector.Enable {
				return nil
			}
			workerInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
				Tracer:            otel.Tracer(serviceName),
				TextMapPropagator: otel.GetTextMapPropagator(),
			})
			if err != nil {
				logger.Fatal("Unable to create worker tracing interceptor", zap.Error(err))
			}
			return []interceptor.WorkerInterceptor{workerInterceptor}
		}(),
	})

	w.RegisterWorkflow(cw.TriggerModelWorkflow)
	w.RegisterActivity(cw.TriggerModelActivity)

	if err := w.Run(worker.InterruptCh()); err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}

	logger.Info("worker is running.")

	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	quitSig := make(chan os.Signal, 1)
	signal.Notify(quitSig, syscall.SIGINT, syscall.SIGTERM)

	// When the server receives a SIGTERM, we'll try to finish ongoing
	// workflows. This is because pipeline trigger activities use a shared
	// in-process memory, which prevents workflows from recovering from
	// interruptions.
	// To handle this properly, we should make activities independent of
	// the shared memory.
	<-quitSig

	time.Sleep(gracefulShutdownWaitPeriod)

	logger.Info("Shutting down worker...")
	w.Stop()
}

func newClients(ctx context.Context, logger *zap.Logger) (
	*redis.Client,
	*gorm.DB,
	ray.Ray,
	temporalclient.Client,
	miniox.Client,
	*repository.InfluxDB,
	func(),
) {
	closeFuncs := map[string]func() error{}

	// Initialize database
	db := database.GetSharedConnection()
	closeFuncs["database"] = func() error {
		database.Close(db)
		return nil
	}

	// Initialize redis client
	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	closeFuncs["redis"] = redisClient.Close

	// Initialize Ray service
	rayService := ray.NewRay(redisClient)
	closeFuncs["ray"] = func() error {
		rayService.Close()
		return nil
	}

	// Initialize Temporal client
	temporalClientOptions, err := temporalx.ClientOptions(config.Config.Temporal, logger)
	if err != nil {
		logger.Fatal("Unable to get Temporal client options", zap.Error(err))
	}

	// Only add interceptor if tracing is enabled
	if config.Config.OTELCollector.Enable {
		temporalTracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
			Tracer:            otel.Tracer(serviceName),
			TextMapPropagator: otel.GetTextMapPropagator(),
		})
		if err != nil {
			logger.Fatal("Unable to create temporal tracing interceptor", zap.Error(err))
		}
		temporalClientOptions.Interceptors = []interceptor.ClientInterceptor{temporalTracingInterceptor}
	}

	temporalClient, err := temporalclient.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal("Unable to create Temporal client", zap.Error(err))
	}
	closeFuncs["temporal"] = func() error {
		temporalClient.Close()
		return nil
	}

	// Initialize MinIO client
	retentionHandler := service.NewRetentionHandler()
	minioClient, err := miniox.NewMinIOClientAndInitBucket(ctx, miniox.ClientParams{
		Config:      config.Config.Minio,
		Logger:      logger,
		ExpiryRules: retentionHandler.ListExpiryRules(),
		AppInfo: miniox.AppInfo{
			Name:    serviceName,
			Version: serviceVersion,
		},
	})
	if err != nil {
		logger.Fatal("failed to create minio client", zap.Error(err))
	}

	// Initialize InfluxDB
	influxDB := repository.MustNewInfluxDB(ctx, config.Config.Server.Debug)
	closeFuncs["influxDB"] = func() error {
		influxDB.Close()
		return nil
	}

	closer := func() {
		for conn, fn := range closeFuncs {
			if err := fn(); err != nil {
				logger.Error("Failed to close conn", zap.Error(err), zap.String("conn", conn))
			}
		}
	}

	return redisClient, db, rayService, temporalClient, minioClient, influxDB, closer
}

func initTemporalNamespace(ctx context.Context, client temporalclient.Client, logger *zap.Logger) {

	resp, err := client.WorkflowService().ListNamespaces(ctx, &workflowservice.ListNamespacesRequest{})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to list namespaces: %s", err))
	}

	found := false
	for _, n := range resp.GetNamespaces() {
		if n.NamespaceInfo.Name == config.Config.Temporal.Namespace {
			found = true
		}
	}

	if !found {
		if _, err := client.WorkflowService().RegisterNamespace(ctx,
			&workflowservice.RegisterNamespaceRequest{
				Namespace: config.Config.Temporal.Namespace,
				WorkflowExecutionRetentionPeriod: func() *durationpb.Duration {
					// Check if the string ends with "d" for day.
					s := config.Config.Temporal.Retention
					if strings.HasSuffix(s, "d") {
						// Parse the number of days.
						days, err := strconv.Atoi(s[:len(s)-1])
						if err != nil {
							logger.Fatal(fmt.Sprintf("Unable to parse retention period in day: %s", err))
						}
						// Convert days to hours and then to a duration.
						return &durationpb.Duration{
							Seconds: int64(time.Duration(days) * 24 * time.Hour / time.Second),
						}
					}
					logger.Fatal(fmt.Sprintf("Unable to parse retention period in day: %s", err))
					return nil
				}(),
			},
		); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to register namespace: %s", err))
		}
	}
}
