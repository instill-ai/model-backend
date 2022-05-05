package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

func upload(c *cli.Context) error {
	filePath := c.String("file")
	modelName := c.String("name")
	if _, err := os.Stat(filePath); err != nil {
		log.Fatalf("File model do not exist, you could download sample-models by scripts/quick-download.sh")
	}

	// Create connection to server with timeout 1000 secs to ensure file streamed successfully
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()
	client := modelPB.NewModelServiceClient(conn)

	streamUploader, err := client.CreateModelBinaryFileUpload(ctx)
	if err != nil {
		log.Fatalf("Could not create a stream to server, please make sure the server is running")
	}
	defer streamUploader.CloseSend() //nolint

	//create a buffer of chunkSize to be streamed
	const chunkSize = 64 * 1024 // 64 KiB
	buf := make([]byte, chunkSize)
	firstChunk := true

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Could not open the file %v", filePath)
	}
	defer file.Close()

	for {
		n, errRead := file.Read(buf)
		if errRead != nil {
			if errRead == io.EOF {
				break
			}
			log.Fatalf("Could not read the file %v", filePath)
		}
		if firstChunk {
			err = streamUploader.Send(&modelPB.CreateModelBinaryFileUploadRequest{
				Model: &modelPB.Model{
					Id: modelName,
				},
				Bytes: buf[:n],
			})
			firstChunk = false
			if err != nil {
				log.Fatalf("Could not send buffer data to server in first chunk")
			}
		} else {
			err = streamUploader.Send(&modelPB.CreateModelBinaryFileUploadRequest{
				Bytes: buf[:n],
			})
			if err != nil {
				log.Fatalf("Could not send buffer data to server")
			}
		}
	}

	res, err := streamUploader.CloseAndRecv()
	if err != nil {
		log.Fatalf("Error %v", err.Error())
	}
	fmt.Println("Created model: ", res)
	return nil
}

func load(c *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()
	client := modelPB.NewModelServiceClient(conn)

	res, err := client.DeployModelInstance(ctx, &modelPB.DeployModelInstanceRequest{
		Name: fmt.Sprintf("models/%v/instances/%v", c.String("name"), c.String("instance")),
	})
	if err != nil {
		log.Fatalf("Could not load model into triton server %v", err.Error())
	}

	fmt.Println("Loaded model: ", res)
	return nil
}

func predict(c *cli.Context) error {
	filePath := c.String("file")
	modelName := c.String("name")
	instanceName := c.String("instance")
	if _, err := os.Stat(filePath); err != nil {
		log.Fatalf("File image do not exist")
	}

	// Set up a connection to the server.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()
	client := modelPB.NewModelServiceClient(conn)

	streamUploader, err := client.TriggerModelInstanceBinaryFileUpload(ctx)
	if err != nil {
		log.Fatalf("Could not create predict stream")
	}
	defer streamUploader.CloseSend() //nolint

	//create a buffer of chunkSize to be streamed
	const chunkSize = 64 * 1024 // 64 KiB
	buf := make([]byte, chunkSize)
	firstChunk := true

	file1, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Could not open the file %v", filePath)
	}
	fi1, _ := file1.Stat()
	defer file1.Close()

	var n int
	var errRead error
	for {
		n, errRead = file1.Read(buf)
		if errRead != nil {
			if errRead == io.EOF {
				break
			} else {
				log.Fatalf("Could not read the file1 %v", filePath)
			}
		}

		if firstChunk {
			err = streamUploader.Send(&modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Name:        fmt.Sprintf("models/%v/instances/%v", modelName, instanceName),
				FileLengths: []uint64{uint64(fi1.Size())},
				Bytes:       buf[:n],
			})
			if err != nil {
				log.Fatalf("Could not send buffer data to server")
			}
			firstChunk = false
		} else {
			err = streamUploader.Send(&modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Bytes: buf[:n],
			})
			if err != nil {
				log.Fatalf("Could not send buffer data to server")
			}
		}
	}

	res, err := streamUploader.CloseAndRecv()
	if err != nil {
		log.Fatalf("Could not predict model %v", err.Error())
	}
	fmt.Println("Predict result: ", res)
	return nil
}

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "upload",
				Aliases: []string{"u"},
				Usage:   "Upload a model file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "Upload model `FILE`",
						FilePath: "./sample-models/yolov4-onnx-cpu.zip",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    "model `NAME`",
						Required: false,
					},
				},
				Action: func(c *cli.Context) error {
					return upload(c)
				},
			},
			{
				Name:    "load",
				Aliases: []string{"l"},
				Usage:   "Load model into Triton Server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    "Model `NAME`",
						Required: true,
					},
					&cli.StringFlag{
						Name:        "instance",
						Aliases:     []string{"i"},
						Usage:       "model `INSTANCE` name",
						DefaultText: "latest",
						Required:    false,
					},
				},
				Action: func(c *cli.Context) error {
					return load(c)
				},
			},
			{
				Name:    "predict",
				Aliases: []string{"p"},
				Usage:   "Predict model",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    "Model `NAME`",
						Required: true,
					},
					&cli.StringFlag{
						Name:        "instance",
						Aliases:     []string{"i"},
						Usage:       "model `INSTANCE` name",
						DefaultText: "latest",
						Required:    false,
					},
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "Upload model `FILE`",
						FilePath: "./sample-models/dog.jpg",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return predict(c)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
