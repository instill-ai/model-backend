package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.temporal.io/sdk/client"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/external"
	"github.com/instill-ai/model-backend/pkg/handler"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/triton"
	"github.com/instill-ai/model-backend/pkg/usage"
	"github.com/instill-ai/model-backend/pkg/util"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/model-backend/pkg/db"
	"github.com/instill-ai/model-backend/pkg/middleware"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

func grpcHandlerFunc(grpcServer *grpc.Server, gwHandler http.Handler, CORSOrigins []string) http.Handler {
	return h2c.NewHandler(
		cors.New(cors.Options{
			AllowedOrigins:   CORSOrigins,
			AllowCredentials: true,
			Debug:            false,
			AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "HEAD"},
		}).Handler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
					grpcServer.ServeHTTP(w, r)
				} else {
					gwHandler.ServeHTTP(w, r)
				}
			})),
		&http2.Server{},
	)
}

func main() {
	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}

	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()
	grpc_zap.ReplaceGrpcLoggerV2(logger)

	db := database.GetConnection()
	defer database.Close(db)

	// Create tls based credential.
	var creds credentials.TransportCredentials
	var err error
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key)
		if err != nil {
			logger.Fatal(fmt.Sprintf("failed to create credentials: %v", err))
		}
	}

	// Shared options for the logger, with a custom gRPC code to log level function.
	opts := []grpc_zap.Option{
		grpc_zap.WithDecider(func(fullMethodName string, err error) bool {
			// will not log gRPC calls if it was a call to liveness or readiness and no error was raised
			if err == nil {
				if match, _ := regexp.MatchString("vdp.model.v1alpha.ModelPublicService/.*ness$", fullMethodName); match {
					return false
				}
			}
			// by default everything will be logged
			return true
		}),
	}

	grpcServerOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			middleware.StreamAppendMetadataInterceptor,
			grpc_zap.StreamServerInterceptor(logger, opts...),
			grpc_recovery.StreamServerInterceptor(middleware.RecoveryInterceptorOpt()),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			middleware.UnaryAppendMetadataInterceptor,
			grpc_zap.UnaryServerInterceptor(logger, opts...),
			grpc_recovery.UnaryServerInterceptor(middleware.RecoveryInterceptorOpt()),
		)),
	}

	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		grpcServerOpts = append(grpcServerOpts, grpc.Creds(creds))
	}

	privateGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(privateGrpcS)

	publicGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(publicGrpcS)

	triton := triton.NewTriton()
	defer triton.Close()

	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient()
	defer mgmtPrivateServiceClientConn.Close()

	pipelinePublicServiceClient, pipelinePublicServiceClientConn := external.InitPipelinePublicServiceClient()
	defer pipelinePublicServiceClientConn.Close()

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	controllerClient, controllerClientConn := external.InitControllerPrivateServiceClient()
	defer controllerClientConn.Close()

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

	temporalClient, err := client.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client: %s", err))
	}
	defer temporalClient.Close()

	repository := repository.NewRepository(db)

	service := service.NewService(repository, triton, mgmtPrivateServiceClient, pipelinePublicServiceClient, redisClient, temporalClient, controllerClient)

	modelPB.RegisterModelPublicServiceServer(
		publicGrpcS,
		handler.NewPublicHandler(service, triton))

	modelPB.RegisterModelPrivateServiceServer(
		privateGrpcS,
		handler.NewPrivateHandler(service, triton))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	privateGwS := runtime.NewServeMux(
		runtime.WithForwardResponseOption(middleware.HttpResponseModifier),
		runtime.WithErrorHandler(middleware.ErrorHandler),
		runtime.WithIncomingHeaderMatcher(middleware.CustomMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   util.MarshalOptions,
			UnmarshalOptions: util.UnmarshalOptions,
		}),
	)

	publicGwS := runtime.NewServeMux(
		runtime.WithForwardResponseOption(middleware.HttpResponseModifier),
		runtime.WithErrorHandler(middleware.ErrorHandler),
		runtime.WithIncomingHeaderMatcher(middleware.CustomMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   util.MarshalOptions,
			UnmarshalOptions: util.UnmarshalOptions,
		}),
	)

	// Register custom route for  POST /v1alpha/models/{name=models/*}/test-multipart which makes model inference for REST multiple-part form-data
	if err := publicGwS.HandlePath("POST", "/v1alpha/{name=models/*}/test-multipart", middleware.AppendCustomHeaderMiddleware(service, handler.HandleTestModelByUpload)); err != nil {
		panic(err)
	}

	// Register custom route for  POST /v1alpha/models/{name=models/*}/trigger-multipart which makes model inference for REST multiple-part form-data
	if err := publicGwS.HandlePath("POST", "/v1alpha/{name=models/*}/trigger-multipart", middleware.AppendCustomHeaderMiddleware(service, handler.HandleTriggerModelByUpload)); err != nil {
		panic(err)
	}

	// Register custom route for  POST /models/multipart which uploads model for REST multiple-part form-data
	if err := publicGwS.HandlePath("POST", "/v1alpha/models/multipart", middleware.AppendCustomHeaderMiddleware(service, handler.HandleCreateModelByMultiPartFormData)); err != nil {
		panic(err)
	}

	// Start usage reporter
	var usg usage.Usage
	if !config.Config.Server.DisableUsage {
		usageServiceClient, usageServiceClientConn := external.InitUsageServiceClient()
		defer usageServiceClientConn.Close()
		logger.Info("try to start usage reporter")
		go func() {
			for {
				usg = usage.NewUsage(ctx, repository, mgmtPrivateServiceClient, redisClient, usageServiceClient)
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
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	} else {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	if err := modelPB.RegisterModelPrivateServiceHandlerFromEndpoint(ctx, privateGwS, fmt.Sprintf(":%v", config.Config.Server.PrivatePort), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}
	if err := modelPB.RegisterModelPublicServiceHandlerFromEndpoint(ctx, publicGwS, fmt.Sprintf(":%v", config.Config.Server.PublicPort), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}

	privateHttpServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", config.Config.Server.PrivatePort),
		Handler: grpcHandlerFunc(privateGrpcS, privateGwS, config.Config.Server.CORSOrigins),
	}

	publicHttpServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", config.Config.Server.PublicPort),
		Handler: grpcHandlerFunc(publicGrpcS, publicGwS, config.Config.Server.CORSOrigins),
	}

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds.
	quitSig := make(chan os.Signal, 1)
	errSig := make(chan error)
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		go func() {
			if err := privateHttpServer.ListenAndServeTLS(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key); err != nil {
				errSig <- err
			}
		}()
	} else {
		go func() {
			if err := privateHttpServer.ListenAndServe(); err != nil {
				errSig <- err
			}
		}()
	}
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		go func() {
			if err := publicHttpServer.ListenAndServeTLS(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key); err != nil {
				errSig <- err
			}
		}()
	} else {
		go func() {
			if err := publicHttpServer.ListenAndServe(); err != nil {
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
		if !config.Config.Server.DisableUsage && usg != nil {
			usg.TriggerSingleReporter(ctx)
		}
		logger.Info("Shutting down server...")
		privateGrpcS.GracefulStop()
		publicGrpcS.GracefulStop()
	}

}
