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
	"github.com/instill-ai/model-backend/internal/external"
	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/internal/util"
	"github.com/instill-ai/model-backend/pkg/handler"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/usage"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/model-backend/internal/db"
	modelWorker "github.com/instill-ai/model-backend/internal/worker"
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
				if match, _ := regexp.MatchString("vdp.model.v1alpha.ModelService/.*ness$", fullMethodName); match {
					return false
				}
			}
			// by default everything will be logged
			return true
		}),
	}

	grpcServerOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamAppendMetadataInterceptor,
			grpc_zap.StreamServerInterceptor(logger, opts...),
			grpc_recovery.StreamServerInterceptor(recoveryInterceptorOpt()),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryAppendMetadataInterceptor,
			grpc_zap.UnaryServerInterceptor(logger, opts...),
			grpc_recovery.UnaryServerInterceptor(recoveryInterceptorOpt()),
		)),
	}

	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		grpcServerOpts = append(grpcServerOpts, grpc.Creds(creds))
	}

	grpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(grpcS)

	triton := triton.NewTriton()
	defer triton.Close()

	userServiceClient, userServiceClientConn := external.InitUserServiceClient()
	defer userServiceClientConn.Close()

	pipelineServiceClient, pipelineServiceClientConn := external.InitPipelineServiceClient()
	defer pipelineServiceClientConn.Close()

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	temporalClient, err := client.Dial(client.Options{
		// ZapAdapter implements log.Logger interface and can be passed
		// to the client constructor using client using client.Options.
		Namespace: modelWorker.Namespace,
		Logger:    zapadapter.NewZapAdapter(logger),
		HostPort:  config.Config.Temporal.ClientOptions.HostPort,
	})
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer temporalClient.Close()

	repository := repository.NewRepository(db)

	modelPB.RegisterModelServiceServer(
		grpcS,
		handler.NewHandler(
			service.NewService(repository, triton, pipelineServiceClient, redisClient, temporalClient),
			triton))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gwS := runtime.NewServeMux(
		runtime.WithForwardResponseOption(httpResponseModifier),
		runtime.WithErrorHandler(errorHandler),
		runtime.WithIncomingHeaderMatcher(customMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   util.MarshalOptions,
			UnmarshalOptions: util.UnmarshalOptions,
		}),
	)

	// Register custom route for  POST /v1alpha/models/{name=models/*/instances/*}/test-multipart which makes model inference for REST multiple-part form-data
	if err := gwS.HandlePath("POST", "/v1alpha/{name=models/*/instances/*}/test-multipart", appendCustomHeaderMiddleware(handler.HandleTestModelInstanceByUpload)); err != nil {
		panic(err)
	}

	// Register custom route for  POST /v1alpha/models/{name=models/*/instances/*}/trigger-multipart which makes model inference for REST multiple-part form-data
	if err := gwS.HandlePath("POST", "/v1alpha/{name=models/*/instances/*}/trigger-multipart", appendCustomHeaderMiddleware(handler.HandleTriggerModelInstanceByUpload)); err != nil {
		panic(err)
	}

	// Register custom route for  POST /models/multipart which uploads model for REST multiple-part form-data
	if err := gwS.HandlePath("POST", "/v1alpha/models/multipart", appendCustomHeaderMiddleware(handler.HandleCreateModelByMultiPartFormData)); err != nil {
		panic(err)
	}

	// Start usage reporter
	var usg usage.Usage
	if !config.Config.Server.DisableUsage {
		usageServiceClient, usageServiceClientConn := external.InitUsageServiceClient()
		defer usageServiceClientConn.Close()
		usg = usage.NewUsage(ctx, repository, userServiceClient, redisClient, usageServiceClient)
		if usg != nil {
			usg.StartReporter(ctx)
		}
	}

	var dialOpts []grpc.DialOption
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	} else {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	if err := modelPB.RegisterModelServiceHandlerFromEndpoint(ctx, gwS, fmt.Sprintf(":%v", config.Config.Server.Port), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", config.Config.Server.Port),
		Handler: grpcHandlerFunc(grpcS, gwS, config.Config.Server.CORSOrigins),
	}

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds.
	quitSig := make(chan os.Signal, 1)
	errSig := make(chan error)
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		go func() {
			if err := httpServer.ListenAndServeTLS(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key); err != nil {
				errSig <- err
			}
		}()
	} else {
		go func() {
			if err := httpServer.ListenAndServe(); err != nil {
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
		grpcS.GracefulStop()
	}

}
