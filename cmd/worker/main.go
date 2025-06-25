package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"

	temporalclient "go.temporal.io/sdk/client"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/x/minio"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/model-backend/pkg/db"
	customlogger "github.com/instill-ai/model-backend/pkg/logger"
	customotel "github.com/instill-ai/model-backend/pkg/logger/otel"
	modelWorker "github.com/instill-ai/model-backend/pkg/worker"
)

var (
	// These variables might be overridden at buildtime.
	serviceVersion = "dev"
	serviceName    = "model-backend-worker"
)

func initTemporalNamespace(ctx context.Context, client temporalclient.Client) {
	logger, _ := customlogger.GetZapLogger(ctx)

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

func main() {

	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	if tp, err := customotel.SetupTracing(ctx, "model-backend-worker"); err != nil {
		panic(err)
	} else {
		defer func() {
			err = tp.Shutdown(ctx)
		}()
	}

	ctx, span := otel.Tracer("worker-tracer").Start(ctx,
		"main",
	)
	defer cancel()

	logger, _ := customlogger.GetZapLogger(ctx)
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	datamodel.InitJSONSchema(ctx)

	db := database.GetSharedConnection()
	defer database.Close(db)

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	rayService := ray.NewRay(redisClient)
	defer rayService.Close()

	temporalTracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
		Tracer:            otel.Tracer("temporal-tracer"),
		TextMapPropagator: b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
	})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create temporal tracing interceptor: %s", err))
	}

	var temporalClientOptions temporalclient.Options
	if config.Config.Temporal.Ca != "" && config.Config.Temporal.Cert != "" && config.Config.Temporal.Key != "" {
		if temporalClientOptions, err = temporal.GetTLSClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger),
			config.Config.Temporal.Ca,
			config.Config.Temporal.Cert,
			config.Config.Temporal.Key,
			config.Config.Temporal.ServerName,
			true,
		); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	} else {
		if temporalClientOptions, err = temporal.GetClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger)); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	}

	temporalClientOptions.Interceptors = []interceptor.ClientInterceptor{temporalTracingInterceptor}
	tempClient, err := temporalclient.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client: %s", err))
	}
	defer tempClient.Close()

	// for only local temporal cluster
	if config.Config.Temporal.Ca == "" && config.Config.Temporal.Cert == "" && config.Config.Temporal.Key == "" {
		initTemporalNamespace(ctx, tempClient)
	}

	// Initialize MinIO client
	retentionHandler := service.NewRetentionHandler()
	minioClient, err := minio.NewMinIOClientAndInitBucket(ctx, minio.ClientParams{
		Config:      config.Config.Minio,
		Logger:      logger,
		ExpiryRules: retentionHandler.ListExpiryRules(),
		AppInfo: minio.AppInfo{
			Name:    serviceName,
			Version: serviceVersion,
		},
	})
	if err != nil {
		logger.Fatal("failed to create minio client", zap.Error(err))
	}

	repo := repository.NewRepository(db, redisClient)
	timeseries := repository.MustNewInfluxDB(ctx, config.Config.Server.Debug)
	defer timeseries.Close()

	cw := modelWorker.NewWorker(redisClient, rayService, repo, timeseries.WriteAPI(), minioClient, nil)

	w := worker.New(tempClient, modelWorker.TaskQueue, worker.Options{})

	w.RegisterWorkflow(cw.TriggerModelWorkflow)
	w.RegisterActivity(cw.TriggerModelActivity)

	span.End()
	if err := w.Run(worker.InterruptCh()); err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}

}
