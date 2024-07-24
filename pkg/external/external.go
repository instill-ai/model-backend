package external

import (
	"context"
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	custom_logger "github.com/instill-ai/model-backend/pkg/logger"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	usagepb "github.com/instill-ai/protogen-go/core/usage/v1beta"
)

func InitMgmtPublicServiceClient(ctx context.Context) (mgmtpb.MgmtPublicServiceClient, *grpc.ClientConn) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.MgmtBackend.HTTPS.Cert != "" && config.Config.MgmtBackend.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.MgmtBackend.HTTPS.Cert, config.Config.MgmtBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PublicPort), clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(config.Config.Server.MaxDataSize), grpc.MaxCallSendMsgSize(config.Config.Server.MaxDataSize)))
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return mgmtpb.NewMgmtPublicServiceClient(clientConn), clientConn
}

// InitMgmtPrivateServiceClient initializes a MgmtPrivateServiceClient instance
func InitMgmtPrivateServiceClient(ctx context.Context) (mgmtpb.MgmtPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := custom_logger.GetZapLogger(ctx)

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

	clientConn, err := grpc.NewClient(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return mgmtpb.NewMgmtPrivateServiceClient(clientConn), clientConn
}

// InitArtifactPrivateServiceClient initializes a ArtifactPrivateServiceClient instance
func InitArtifactPrivateServiceClient(ctx context.Context) (artifactpb.ArtifactPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if config.Config.ArtifactBackend.HTTPS.Cert != "" && config.Config.ArtifactBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.ArtifactBackend.HTTPS.Cert, config.Config.ArtifactBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(fmt.Sprintf("%v:%v", config.Config.ArtifactBackend.Host, config.Config.ArtifactBackend.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return artifactpb.NewArtifactPrivateServiceClient(clientConn), clientConn
}

// InitUsageServiceClient initializes a UsageServiceClient instance (no mTLS)
func InitUsageServiceClient(ctx context.Context) (usagepb.UsageServiceClient, *grpc.ClientConn) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var err error
	if config.Config.Server.Usage.TLSEnabled {
		tlsConfig := &tls.Config{}
		clientDialOpts = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(
		fmt.Sprintf("%v:%v", config.Config.Server.Usage.Host, config.Config.Server.Usage.Port),
		clientDialOpts, grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.Config.Server.MaxDataSize*constant.MB),
			grpc.MaxCallSendMsgSize(config.Config.Server.MaxDataSize*constant.MB),
		),
	)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return usagepb.NewUsageServiceClient(clientConn), clientConn
}
