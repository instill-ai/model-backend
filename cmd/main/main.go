package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	openfga "github.com/openfga/api/proto/openfga/v1"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/handler"
	"github.com/instill-ai/model-backend/pkg/middleware"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/usage"
	"github.com/instill-ai/x/client"
	"github.com/instill-ai/x/server/grpc/gateway"
	"github.com/instill-ai/x/temporal"

	database "github.com/instill-ai/model-backend/pkg/db"
	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	grpcclientx "github.com/instill-ai/x/client/grpc"
	logx "github.com/instill-ai/x/log"
	miniox "github.com/instill-ai/x/minio"
	otelx "github.com/instill-ai/x/otel"
	grpcx "github.com/instill-ai/x/server/grpc"
)

var (
	// These variables might be overridden at buildtime.
	serviceName    = "model-backend"
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
		otelx.WithHost(config.Config.OtelCollector.Host),
		otelx.WithPort(config.Config.OtelCollector.Port),
		otelx.WithCollectorEnable(config.Config.OtelCollector.Enable),
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

	// Get gRPC server options and credentials
	grpcServerOpts, creds, err := grpcx.NewGRPCOptionAndCreds(
		grpcx.WithServiceName(serviceName),
		grpcx.WithServiceVersion(serviceVersion),
		grpcx.WithServiceConfig(client.HTTPSConfig{
			Cert: config.Config.Server.HTTPS.Cert,
			Key:  config.Config.Server.HTTPS.Key,
		}),
		grpcx.WithOTELCollectorEnable(config.Config.OtelCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create gRPC server options and credentials", zap.Error(err))
	}

	privateGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(privateGrpcS)

	publicGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(publicGrpcS)

	// Initialize all clients
	mgmtPrivateServiceClient, artifactPrivateServiceClient, redisClient, db,
		minioClient, aclClient, rayService, temporalClient, influxDB, closeClients := newClients(ctx, logger)
	defer closeClients()

	repo := repository.NewRepository(db, redisClient)
	service := service.NewService(
		repo,
		influxDB.WriteAPI(),
		mgmtPrivateServiceClient,
		artifactPrivateServiceClient,
		redisClient,
		temporalClient,
		rayService,
		aclClient,
		minioClient,
		service.NewRetentionHandler(),
		config.Config.Server.InstillCoreHost,
	)

	modelpb.RegisterModelPublicServiceServer(
		publicGrpcS,
		handler.NewPublicHandler(ctx, service, rayService, nil))

	modelpb.RegisterModelPrivateServiceServer(
		privateGrpcS,
		handler.NewPrivateHandler(ctx, service))

	privateServeMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(gateway.HTTPResponseModifier),
		runtime.WithErrorHandler(gateway.ErrorHandler),
		runtime.WithIncomingHeaderMatcher(gateway.CustomHeaderMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseEnumNumbers:  false,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	publicServeMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(gateway.HTTPResponseModifier),
		runtime.WithErrorHandler(gateway.ErrorHandler),
		runtime.WithIncomingHeaderMatcher(gateway.CustomHeaderMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseEnumNumbers:  false,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	// Register custom route for REST trigger multipart form-data
	// TODO: combine multipart trigger with /trigger like pipeline-backend
	if err := publicServeMux.HandlePath("POST", "/v1alpha/{path=users/*/models/*}/versions/{version=*}/trigger-multipart", middleware.AppendCustomHeaderMiddleware(service, repo, handler.HandleTriggerMultipartForm)); err != nil {
		panic(err)
	}
	if err := publicServeMux.HandlePath("POST", "/v1alpha/{path=organizations/*/models/*}/versions/{version=*}/trigger-multipart", middleware.AppendCustomHeaderMiddleware(service, repo, handler.HandleTriggerMultipartForm)); err != nil {
		panic(err)
	}
	if err := publicServeMux.HandlePath("POST", "/v1alpha/{path=namespaces/*/models/*}/versions/{version=*}/trigger-multipart", middleware.AppendCustomHeaderMiddleware(service, repo, handler.HandleTriggerMultipartForm)); err != nil {
		panic(err)
	}
	if err := publicServeMux.HandlePath("POST", "/v1alpha/{path=users/*/models/*}/trigger-multipart", middleware.AppendCustomHeaderMiddleware(service, repo, handler.HandleTriggerMultipartForm)); err != nil {
		panic(err)
	}
	if err := publicServeMux.HandlePath("POST", "/v1alpha/{path=organizations/*/models/*}/trigger-multipart", middleware.AppendCustomHeaderMiddleware(service, repo, handler.HandleTriggerMultipartForm)); err != nil {
		panic(err)
	}
	if err := publicServeMux.HandlePath("POST", "/v1alpha/{path=namespaces/*/models/*}/trigger-multipart", middleware.AppendCustomHeaderMiddleware(service, repo, handler.HandleTriggerMultipartForm)); err != nil {
		panic(err)
	}

	if err := publicServeMux.HandlePath("GET", "/v1alpha/{path=users/*/models/*}/image", middleware.AppendCustomHeaderMiddleware(service, repo, middleware.HandleProfileImage)); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("GET", "/v1alpha/{path=organizations/*/models/*}/image", middleware.AppendCustomHeaderMiddleware(service, repo, middleware.HandleProfileImage)); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("GET", "/v1alpha/{path=namespaces/*/models/*}/image", middleware.AppendCustomHeaderMiddleware(service, repo, middleware.HandleProfileImage)); err != nil {
		logger.Fatal(err.Error())
	}

	// Start usage reporter
	var usg usage.Usage
	if config.Config.Server.Usage.Enabled {
		usageServiceClient, usageServiceClientConn := usage.InitUsageServiceClient(ctx)
		defer usageServiceClientConn.Close()
		logger.Info("try to start usage reporter")
		go func() {
			for {
				usg = usage.NewUsage(ctx, repo, mgmtPrivateServiceClient, redisClient, usageServiceClient, serviceVersion)
				if usg != nil {
					usg.StartReporter(ctx)
					logger.Info("usage reporter started")
					break
				}
				logger.Warn("retry to start usage reporter after 5 minutes")
				time.Sleep(5 * time.Minute)
			}
		}()
	}

	var dialOpts []grpc.DialOption
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(creds), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(client.MaxPayloadSize), grpc.MaxCallSendMsgSize(client.MaxPayloadSize))}
	} else {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(client.MaxPayloadSize), grpc.MaxCallSendMsgSize(client.MaxPayloadSize))}
	}

	if err := modelpb.RegisterModelPrivateServiceHandlerFromEndpoint(ctx, privateServeMux, fmt.Sprintf(":%v", config.Config.Server.PrivatePort), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}
	if err := modelpb.RegisterModelPublicServiceHandlerFromEndpoint(ctx, publicServeMux, fmt.Sprintf(":%v", config.Config.Server.PublicPort), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}

	privateHTTPServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", config.Config.Server.PrivatePort),
		Handler: grpcHandlerFunc(privateGrpcS, privateServeMux),
	}

	publicHTTPServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", config.Config.Server.PublicPort),
		Handler: grpcHandlerFunc(publicGrpcS, publicServeMux),
	}

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds.
	quitSig := make(chan os.Signal, 1)
	errSig := make(chan error)
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		go func() {
			if err := privateHTTPServer.ListenAndServeTLS(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key); err != nil {
				errSig <- err
			}
		}()
	} else {
		go func() {
			if err := privateHTTPServer.ListenAndServe(); err != nil {
				errSig <- err
			}
		}()
	}
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		go func() {
			if err := publicHTTPServer.ListenAndServeTLS(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key); err != nil {
				errSig <- err
			}
		}()
	} else {
		go func() {
			if err := publicHTTPServer.ListenAndServe(); err != nil {
				errSig <- err
			}
		}()
	}

	logger.Info("gRPC server is running.")

	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quitSig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errSig:
		logger.Error(fmt.Sprintf("Fatal error: %v\n", err))
	case <-quitSig:
		// send out the usage report at exit
		if config.Config.Server.Usage.Enabled && usg != nil {
			usg.TriggerSingleReporter(ctx)
		}
		logger.Info("Shutting down server...")
		privateGrpcS.GracefulStop()
		publicGrpcS.GracefulStop()
	}

}

func grpcHandlerFunc(grpcServer *grpc.Server, gwHandler http.Handler) http.Handler {
	return h2c.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
				grpcServer.ServeHTTP(w, r)
			} else {
				gwHandler.ServeHTTP(w, r)
			}
		}),
		&http2.Server{},
	)
}

func newClients(ctx context.Context, logger *zap.Logger) (
	mgmtpb.MgmtPrivateServiceClient,
	artifactpb.ArtifactPrivateServiceClient,
	*redis.Client,
	*gorm.DB,
	miniox.Client,
	*acl.ACLClient,
	ray.Ray,
	temporalclient.Client,
	*repository.InfluxDB,
	func(),
) {
	closeFuncs := map[string]func() error{}

	// Initialize mgmt private service client
	mgmtPrivateServiceClient, mgmtPrivateClose, err := grpcclientx.NewMgmtPrivateClient(config.Config.MgmtBackend)
	if err != nil {
		logger.Fatal("failed to create mgmt private service client", zap.Error(err))
	}
	closeFuncs["mgmtPrivate"] = mgmtPrivateClose

	// Initialize artifact private service client
	artifactPrivateServiceClient, artifactPrivateClose, err := grpcclientx.NewArtifactPrivateClient(config.Config.ArtifactBackend)
	if err != nil {
		logger.Fatal("failed to create artifact private service client", zap.Error(err))
	}
	closeFuncs["artifactPrivate"] = artifactPrivateClose

	// Initialize database
	db := database.GetSharedConnection()
	closeFuncs["database"] = func() error {
		database.Close(db)
		return nil
	}

	// Initialize redis client
	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	closeFuncs["redis"] = redisClient.Close

	// Initialize MinIO client
	minioClient, err := miniox.NewMinIOClientAndInitBucket(ctx, miniox.ClientParams{
		Config:      config.Config.Minio,
		Logger:      logger,
		ExpiryRules: service.NewRetentionHandler().ListExpiryRules(),
		AppInfo: miniox.AppInfo{
			Name:    serviceName,
			Version: serviceVersion,
		},
	})
	if err != nil {
		logger.Fatal("failed to create minio client", zap.Error(err))
	}

	// Initialize ACL client
	fgaClient, fgaClientConn := acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Host, config.Config.OpenFGA.Port)
	if fgaClientConn != nil {
		closeFuncs["fga"] = fgaClientConn.Close
	}

	var fgaReplicaClient openfga.OpenFGAServiceClient
	if config.Config.OpenFGA.Replica.Host != "" {
		var fgaReplicaClientConn *grpc.ClientConn
		fgaReplicaClient, fgaReplicaClientConn = acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Replica.Host, config.Config.OpenFGA.Replica.Port)
		if fgaReplicaClientConn != nil {
			closeFuncs["fgaReplica"] = fgaReplicaClientConn.Close
		}
	}

	aclClient := acl.NewACLClient(fgaClient, fgaReplicaClient, redisClient)

	// Initialize Ray service
	rayService := ray.NewRay(redisClient)
	closeFuncs["ray"] = rayService.Close

	// Initialize Temporal client
	temporalTracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
		Tracer:            otel.Tracer(serviceName + "-temporal"),
		TextMapPropagator: otel.GetTextMapPropagator(),
	})
	if err != nil {
		logger.Fatal("Unable to create temporal tracing interceptor", zap.Error(err))
	}

	temporalClientOptions, err := temporal.ClientOptions(config.Config.Temporal, logger)
	if err != nil {
		logger.Fatal("Unable to get Temporal client options", zap.Error(err))
	}

	temporalClientOptions.Interceptors = []interceptor.ClientInterceptor{temporalTracingInterceptor}
	temporalClient, err := temporalclient.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal("Unable to create client", zap.Error(err))
	}
	closeFuncs["temporal"] = func() error {
		temporalClient.Close()
		return nil
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

	return mgmtPrivateServiceClient, artifactPrivateServiceClient, redisClient, db, minioClient, aclClient, rayService, temporalClient, influxDB, closer
}
