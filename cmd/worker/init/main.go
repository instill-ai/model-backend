package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/x/zapadapter"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	"github.com/instill-ai/model-backend/internal/worker"
)

func main() {
	logger, _ := logger.GetZapLogger()

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}

	// The client is a heavyweight object that should be created once per process.
	c, err := client.NewNamespaceClient(client.Options{
		Logger:   zapadapter.NewZapAdapter(logger),
		HostPort: config.Config.Temporal.ClientOptions.HostPort,
	})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client %s", err.Error()))
	}
	defer c.Close()

	retention := time.Duration(24 * time.Hour)
	req := &workflowservice.RegisterNamespaceRequest{
		Namespace:                        worker.Namespace,
		Description:                      "model backend name space ",
		WorkflowExecutionRetentionPeriod: &retention,
	}
	if err = c.Register(context.Background(), req); err != nil {
		if !strings.Contains(string(err.Error()), "already exists") {
			logger.Fatal(fmt.Sprintf("Unable to register namespace %s", err.Error()))
		} else {
			logger.Info("Namespace already existed, skip creating")
		}
	}
}
