package external

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/logger"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
)

// InitUserServiceClient initialises a UserServiceClient instance
func InitUserServiceClient() (mgmtPB.UserServiceClient, *grpc.ClientConn) {
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

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.Port), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return mgmtPB.NewUserServiceClient(clientConn), clientConn
}

// InitUsageServiceClient initialises a UsageServiceClient instance
func InitUsageServiceClient() (usagePB.UsageServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger()

	var clientDialOpts grpc.DialOption
	var usageCreds credentials.TransportCredentials
	var err error
	if config.Config.UsageBackend.HTTPS.Cert != "" && config.Config.UsageBackend.HTTPS.Key != "" {
		usageCreds, err = credentials.NewServerTLSFromFile(config.Config.UsageBackend.HTTPS.Cert, config.Config.UsageBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(usageCreds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.UsageBackend.Host, config.Config.UsageBackend.Port), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return usagePB.NewUsageServiceClient(clientConn), clientConn

}
