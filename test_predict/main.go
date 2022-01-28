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
	// imagePath := "/Users/nguyenvantam/Desktop/working/INSTILL/src/client/src/python/examples/qa/images/vulture.jpeg"
	imagePath := "/Users/nguyenvantam/Desktop/drive-download-20220121T191023Z-001/bnb_3.jpg"
	file, errOpen := os.Open(imagePath)
	fmt.Println("runnnnn ", imagePath)
	if errOpen != nil {
		errOpen = errors.Wrapf(errOpen,
			"failed to open file")
		return
	}

	defer file.Close()

	// Set up a connection to the server.
	// conn, err := grpc.Dial("0.0.0.0:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.Dial("localhost:8080", grpc.WithTimeout(30*time.Second), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := model.NewModelClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	streamUploader, err := c.PredictModel(ctx)
	defer streamUploader.CloseSend()

	//create a buffer of chunkSize to be streamed
	const chunkSize = 64 * 1024 // 64 KiB
	buf := make([]byte, chunkSize)
	firstChunk := true

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
			err = streamUploader.Send(&model.PredictModelRequest{
				// ModelId:      "inception_graphdef",
				ModelId:      "essemble",
				ModelVersion: 1,
				ModelType:    "classification",
				Content:      buf[:n],
			})
			firstChunk = false
		} else {
			err = streamUploader.Send(&model.PredictModelRequest{
				Content: buf[:n],
			})
		}
	}

	res, err := streamUploader.CloseAndRecv()
	fmt.Println("Predict: ", res, err)
}
