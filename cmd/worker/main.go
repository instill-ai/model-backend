package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v9"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	openfgaClient "github.com/openfga/go-sdk/client"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/external"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/triton"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/model-backend/pkg/db"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	modelWorker "github.com/instill-ai/model-backend/pkg/worker"
)

func main() {

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	if tp, err := custom_otel.SetupTracing(ctx, "model-backend-worker"); err != nil {
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

	logger, _ := logger.GetZapLogger(ctx)
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	db := database.GetSharedConnection()
	defer database.Close(db)

	triton := triton.NewTriton()
	defer triton.Close()

	controllerClient, controllerClientConn := external.InitControllerPrivateServiceClient(ctx)
	defer controllerClientConn.Close()

	rayService := ray.NewRay()
	defer rayService.Close()

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	temporalTracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
		Tracer:            otel.Tracer("temporal-tracer"),
		TextMapPropagator: b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
	})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create temporal tracing interceptor: %s", err))
	}
	var temporalClientOptions client.Options
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
	temporalClient, err := client.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client: %s", err))
	}
	defer temporalClient.Close()

	fgaClient, err := openfgaClient.NewSdkClient(&openfgaClient.ClientConfiguration{
		ApiScheme: "http",
		ApiHost:   fmt.Sprintf("%s:%d", config.Config.OpenFGA.Host, config.Config.OpenFGA.Port),
	})

	if err != nil {
		panic(err)
	}

	var aclClient acl.ACLClient
	if stores, err := fgaClient.ListStores(context.Background()).Execute(); err == nil {
		fgaClient.SetStoreId(stores.Stores[0].Id)
		if models, err := fgaClient.ReadAuthorizationModels(context.Background()).Execute(); err == nil {
			aclClient = acl.NewACLClient(fgaClient, &models.AuthorizationModels[0].Id)
		}
		if err != nil {
			panic(err)
		}

	} else {
		panic(err)
	}

	cw := modelWorker.NewWorker(repository.NewRepository(db), redisClient, triton, controllerClient, rayService, &aclClient)

	w := worker.New(temporalClient, modelWorker.TaskQueue, worker.Options{})

	w.RegisterWorkflow(cw.DeployModelWorkflow)
	w.RegisterActivity(cw.DeployModelActivity)
	w.RegisterWorkflow(cw.UnDeployModelWorkflow)
	w.RegisterActivity(cw.UnDeployModelActivity)
	w.RegisterWorkflow(cw.CreateModelWorkflow)

	span.End()
	if err := w.Run(worker.InterruptCh()); err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}

}
