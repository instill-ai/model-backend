package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/model-backend/internal/db"
	modelWorker "github.com/instill-ai/model-backend/internal/worker"
)

func initialize() {

	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	runCmd := exec.CommandContext(context.Background(),
		"docker",
		"exec",
		"temporal-admin-tools",
		"/bin/bash",
		"-c",
		`tctl --auto_confirm admin cluster add-search-attributes \
			--name Type --type Text --name ModelUID --type Text \
			--name ModelInstanceUID --type Text --name Owner --type Text`,
	)

	var out bytes.Buffer
	runCmd.Stdout = &out
	runCmd.Stderr = &out

	if err := runCmd.Run(); err != nil {
		logger.Debug(err.Error())
	}

	logger.Info(fmt.Sprintf("Docker exec tctl - add search attributes: %s", out.String()))
	out.Reset()
}

func main() {
	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	if err := config.Init(); err != nil {
		logger.Fatal(err.Error())
	}
	logger.Info("Config initialized")
	initialize()
	logger.Info("Initialization finished")

	db := database.GetConnection()
	defer database.Close(db)

	triton := triton.NewTriton()
	defer triton.Close()

	clientNamespace, err := client.NewNamespaceClient(client.Options{
		HostPort: config.Config.Temporal.ClientOptions.HostPort,
	})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create namespace client: %s", err))
	}
	defer clientNamespace.Close()

	retention := time.Duration(24 * time.Hour)
	if err = clientNamespace.Register(context.Background(), &workflowservice.RegisterNamespaceRequest{
		Namespace:                        modelWorker.Namespace,
		Description:                      "For workflows triggered in the model-backend",
		OwnerEmail:                       "infra@instill.tech",
		WorkflowExecutionRetentionPeriod: &retention,
	}); err != nil {
		logger.Error(fmt.Sprintf("Unable to register namespace: %s", err))
	}

	for start := time.Now(); time.Since(start) < time.Second*30; {
		_, err := clientNamespace.Describe(context.Background(), modelWorker.Namespace)
		_, ok := err.(*serviceerror.NamespaceNotFound)
		if !ok {
			break
		}
		time.Sleep(time.Second * 1)
	}

	cw := modelWorker.NewWorker(repository.NewRepository(db), triton)

	c, err := client.Dial(client.Options{
		// ZapAdapter implements log.Logger interface and can be passed
		// to the client constructor using client using client.Options.
		Logger:    zapadapter.NewZapAdapter(logger),
		HostPort:  config.Config.Temporal.ClientOptions.HostPort,
		Namespace: modelWorker.Namespace,
	})

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
	w.RegisterWorkflow(cw.SearchAttributeReadyWorkflow)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}
}
