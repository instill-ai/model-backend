package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"

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
		"tctl --namespace model-backend namespace register",
	)

	var out bytes.Buffer
	runCmd.Stdout = &out
	runCmd.Stderr = &out

	if err := runCmd.Run(); err != nil {
		logger.Debug(err.Error())
	}

	logger.Info(fmt.Sprintf("Docker exec tctl - add namespace registration model-backend: %s", out.String()))
	out.Reset()

	runCmd = exec.CommandContext(context.Background(),
		"docker",
		"exec",
		"temporal-admin-tools",
		"/bin/bash",
		"-c",
		`tctl --auto_confirm admin cluster add-search-attributes \
			--name Type --type Text --name ModelUID --type Text \
			--name ModelInstanceUID --type Text --name Owner --type Text`,
	)

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

	cw := modelWorker.NewWorker(repository.NewRepository(db), triton)

	c, err := client.Dial(client.Options{
		// ZapAdapter implements log.Logger interface and can be passed
		// to the client constructor using client using client.Options.
		Logger:    zapadapter.NewZapAdapter(logger),
		HostPort:  config.Config.Temporal.ClientOptions.HostPort,
		Namespace: "model-backend",
	})

	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, modelWorker.TaskQueue, worker.Options{})

	w.RegisterWorkflow(cw.DeployModelWorkflow)
	w.RegisterActivity(cw.DeployModelActivity)
	w.RegisterWorkflow(cw.UnDeployModelWorkflow)
	w.RegisterActivity(cw.UnDeployModelActivity)
	w.RegisterWorkflow(cw.CreateModelWorkflow)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker %s", err))
	}
}
