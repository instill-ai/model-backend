package external

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/logger"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
)

// InitMgmtAdminServiceClient initialises a MgmtAdminServiceClient instance
func InitMgmtAdminServiceClient() (mgmtPB.MgmtAdminServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger()

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if config.Config.MgmtBackend.HTTPS.Cert != "" && config.Config.MgmtBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.MgmtBackend.HTTPS.Cert, config.Config.MgmtBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.AdminHost, config.Config.MgmtBackend.AdminPort), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return mgmtPB.NewMgmtAdminServiceClient(clientConn), clientConn
}

// InitUsageServiceClient initializes a UsageServiceClient instance
func InitUsageServiceClient() (usagePB.UsageServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger()

	var clientDialOpts grpc.DialOption
	if config.Config.UsageServer.TLSEnabled {
		roots, err := x509.SystemCertPool()
		if err != nil {
			logger.Fatal(err.Error())
		}

		tlsConfig := tls.Config{
			RootCAs:            roots,
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2"},
		}
		clientDialOpts = grpc.WithTransportCredentials(credentials.NewTLS(&tlsConfig))
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(
		fmt.Sprintf("%v:%v", config.Config.UsageServer.Host, config.Config.UsageServer.Port),
		clientDialOpts,
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  500 * time.Millisecond,
				Multiplier: 1.5,
				Jitter:     0.2,
				MaxDelay:   19 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
	)

	if err != nil {
		logger.Fatal(err.Error())
	}

	return usagePB.NewUsageServiceClient(clientConn), clientConn
}

// InitPipelineServiceClient initialises a PipelineServiceClient instance
func InitPipelineServiceClient() (pipelinePB.PipelineServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger()

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if config.Config.PipelineBackend.HTTPS.Cert != "" && config.Config.PipelineBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.PipelineBackend.HTTPS.Cert, config.Config.PipelineBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.PipelineBackend.Host, config.Config.PipelineBackend.Port), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return pipelinePB.NewPipelineServiceClient(clientConn), clientConn
}
