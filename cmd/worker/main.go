package main

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/external"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/triton"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/model-backend/pkg/db"
	modelWorker "github.com/instill-ai/model-backend/pkg/worker"
)
g
func main() {
	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	if err := config.Init(); err != nil {
		logger.Fatal(err.Error())
	}

	db := database.GetConnection()
	defer database.Close(db)

	triton := triton.NewTriton()
	defer triton.Close()

	controllerCLient, controllerCLientConn := external.InitControllerPrivateServiceClient()
	defer controllerCLientConn.Close()

	clientNamespace, err := client.NewNamespaceClient(client.Options{
		HostPort: config.Config.Temporal.ClientOptions.HostPort,
	})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create namespace client: %s", err))
	}
	defer clientNamespace.Close()

	retention := time.Duration(72 * time.Hour)
	if err = clientNamespace.Register(context.Background(), &workflowservice.RegisterNamespaceRequest{
		Namespace:                        modelWorker.Namespace,
		Description:                      "For workflows triggered in the model-backend",
		OwnerEmail:                       "infra@instill.tech",
		WorkflowExecutionRetentionPeriod: &retention,
	}); err != nil {
		if _, ok := err.(*serviceerror.NamespaceAlreadyExists); !ok {
			logger.Error(fmt.Sprintf("Unable to register namespace: %s", err))
		}
	}

	cw := modelWorker.NewWorker(repository.NewRepository(db), triton, controllerCLient)

	c, err := client.Dial(client.Options{
		// ZapAdapter implements log.Logger interface and can be passed
		// to the client constructor using client using client.Options.
		Logger:    zapadapter.NewZapAdapter(logger),
		HostPort:  config.Config.Temporal.ClientOptions.HostPort,
		Namespace: modelWorker.Namespace,
	})

	// Note that Namespace registration using this API takes up to 10 seconds to complete.
	// Ensure to wait for this registration to complete before starting the Workflow Execution against the Namespace.
	time.Sleep(time.Second * 10)

	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client: %s", err))
	}
	defer c.Close()

	w := worker.New(c, modelWorker.TaskQueue, worker.Options{})

	w.RegisterWorkflow(cw.DeployModelWorkflow)
	w.RegisterActivity(cw.DeployModelActivity)
	w.RegisterWorkflow(cw.UnDeployModelWorkflow)
	w.RegisterActivity(cw.UnDeployModelActivity)
	w.RegisterWorkflow(cw.CreateModelWorkflow)
	w.RegisterWorkflow(cw.AddSearchAttributeWorkflow)

	if err := w.Run(worker.InterruptCh()); err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}

}
