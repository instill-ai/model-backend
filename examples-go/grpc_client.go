package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/instill-ai/protogen-go/model"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func upload(c *cli.Context) error {
	filePath := c.String("file")
	if _, err := os.Stat(filePath); err != nil {
		log.Fatalf("File model do not exist, you could download sample-models by examples-go/quick-download.sh")
	}

	// Create connection to server with timeout 120 secs to ensure file streamed successfully
	conn, err := grpc.Dial("localhost:8080", grpc.WithTimeout(120*time.Second), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()
	client := model.NewModelClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	streamUploader, err := client.CreateModelByUpload(ctx)
	if err != nil {
		log.Fatalf("Could not create a stream to server, please make sure the server is running")
	}
	defer streamUploader.CloseSend()

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
			err = streamUploader.Send(&model.CreateModelRequest{
				Description: "Description",
				Type:        "tensorrt",
				Framework:   "pytorch",
				Optimized:   false,
				Visibility:  "public",
				Content:     buf[:n],
			})
			firstChunk = false
			if err != nil {
				log.Fatalf("Could not send buffer data to server")
			}
		} else {
			err = streamUploader.Send(&model.CreateModelRequest{
				Content: buf[:n],
			})
			if err != nil {
				log.Fatalf("Could not send buffer data to server")
			}
		}
	}

	res, err := streamUploader.CloseAndRecv()
	if err != nil {
		log.Fatalf("Could not get response from server")
	}
	fmt.Println("Created model: ", res)
	return nil
}

func load(c *cli.Context) error {
	conn, err := grpc.Dial("localhost:8080", grpc.WithTimeout(30*time.Second), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()
	client := model.NewModelClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	res, err := client.UpdateModel(ctx, &model.UpdateModelRequest{
		Model: &model.UpdateModelInfo{
			Name:   c.String("name"),
			Status: 1, // 0: unload model from triton server, 1: load model into triton server
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"name", "status"},
		},
	})
	if err != nil {
		log.Fatalf("Could not load model into triton server %v", err.Error())
	}

	fmt.Println("Loaded model: ", res)
	return nil
}

func predict(c *cli.Context) error {
	filePath := c.String("file")
	if _, err := os.Stat(filePath); err != nil {
		log.Fatalf("File image do not exist")
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:8080", grpc.WithTimeout(60*time.Second), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()
	client := model.NewModelClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	streamUploader, err := client.PredictModelByUpload(ctx)
	defer streamUploader.CloseSend()

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
			err = streamUploader.Send(&model.PredictModelRequest{
				Name:    "ensemble",
				Version: 1,
				Type:    0,
				Content: buf[:n],
			})
			if err != nil {
				log.Fatalf("Could not send buffer data to server")
			}
			firstChunk = false
		} else {
			err = streamUploader.Send(&model.PredictModelRequest{
				Content: buf[:n],
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
						Name:     "type",
						Aliases:  []string{"y"},
						Usage:    "Model `TYPE`",
						Required: true,
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
