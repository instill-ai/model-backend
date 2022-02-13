# Model Backend <!-- omit in toc -->

The service serve for uploading AI model into Instill platform and retrieving AI model info.

Table of Contents:
- [Prerequisite](#prerequisite)
- [Quick start](#quick-start)
- [In the nut shell](#in-the-nut-shell)
- [Logging with Logrus](#logging-with-logrus)
  - [Gin logger](#gin-logger)
  - [Debug logger](#debug-logger)
- [How to contribute](#how-to-contribute)
  - [Commit changes and submit PRs](#commit-changes-and-submit-prs)
- [CI/CD](#cicd)
  - [Push](#push)
  - [Pull request](#pull-request)
  - [Image purge](#image-purge)

## Prerequisite
a 
- [conda-pack](https://artifacts.instill.tech/visual-data-preparation/conda-pack) this is a Conda environment for Python Backend running in Triton server 
- [sample models](https://artifacts.instill.tech/visual-data-preparation/sample-models/yolov4-onnx-cpu.zip) example CPU models running in Triton server
To download those dependencies, you could run quick-download.sh
```bash
$ ./examples-go/quick-download.sh
```

## Quick start

```bash
$ docker-compose up -d
$ go run ./examples-go/grpc_client.go upload -f sample-models/yolov4-onnx-cpu.zip # upload model file
$ go run ./examples-go/grpc_client.go load -n ensemble
$ go run ./examples-go/grpc_client.go predict -n ensemble -t 1 -f sample-models/dog.jpg # -t 0: classification model 1: object detection model
```

Note: check the server running via `curl`:

```bash
$ curl http://localhost:8080/__health
{"status":"ok"}%

$ curl http://localhost:8080/v1/helloworld
{"error":"You've not implemented this API yet"}%
```
