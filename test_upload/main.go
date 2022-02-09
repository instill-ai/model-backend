// Package main implements a client for Model service.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/instill-ai/model-backend/internal-protogen-go/model"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Set up a connection to the server.
	// conn, err := grpc.Dial("0.0.0.0:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.Dial("localhost:8080", grpc.WithTimeout(120*time.Second), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := model.NewModelClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	streamUploader, err := c.CreateModel(ctx)
	defer streamUploader.CloseSend()

	//create a buffer of chunkSize to be streamed
	const chunkSize = 64 * 1024 // 64 KiB
	buf := make([]byte, chunkSize)
	firstChunk := true

	file, errOpen := os.Open("/Users/nguyenvantam/Desktop/test_model/Archive.zip")
	// file, errOpen := os.Open("/Users/nguyenvantam/Desktop/triton-models/number-onnx-cpu/Archive.zip")
	// file, errOpen := os.Open("/Users/nguyenvantam/Desktop/working/INSTILL/src/model-backend/inception_graphdef.zip")
	if errOpen != nil {
		errOpen = errors.Wrapf(errOpen,
			"failed to open file")
		return
	}

	defer file.Close()

	for {
		n, errRead := file.Read(buf)
		if errRead != nil {
			if errRead == io.EOF {
				errRead = nil
				break
			}

			errRead = errors.Wrapf(errRead,
				"errored while copying from file to buf")
			return
		}
		if firstChunk {
			err = streamUploader.Send(&model.CreateModelRequest{
				Name:        "yolov4",
				Description: "Description",
				Type:        "tensorrt",
				Framework:   "pytorch",
				Optimized:   false,
				Visibility:  "public",
				Filename:    "aa.zip",
				Content:     buf[:n],
			})
			firstChunk = false
		} else {
			err = streamUploader.Send(&model.CreateModelRequest{
				Content: buf[:n],
			})
		}
	}

	res, err := streamUploader.CloseAndRecv()
	fmt.Println("Created model: ", res, err)
}
