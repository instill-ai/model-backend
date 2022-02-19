# Model Backend <!-- omit in toc -->

The service serve for uploading AI model into Instill platform and retrieving AI model info.

Table of Contents:
- [Prerequisite](#prerequisite)
- [Quick start](#quick-start)
- [Community support](#community-support)
- [Documentation](#documentation)
  - [API reference](#api-reference)
  - [Build docker](#build-docker)
- [License](#license)


## Prerequisite
- [conda-pack](https://artifacts.instill.tech/visual-data-preparation/conda-pack) this is a Conda environment for Python Backend running in Triton server 
- [sample models](https://artifacts.instill.tech/visual-data-preparation/sample-models/yolov4-onnx-cpu.zip) example CPU models running in Triton server
To download those dependencies, you could run quick-download.sh
```bash
$ ./examples-go/quick-download.sh
```

## Quick start

```bash
$ docker-compose up -d
$ go run ./examples-go/grpc_client.go upload --file sample-models/yolov4-onnx-cpu.zip --name yolov4 --cvtask 2  # upload a YOLOv4 model for object detection; note --cvtask 0: undefined 1: image classification task and 2: object detection task 
$ go run ./examples-go/grpc_client.go load -n yolov4 --version 1  # deploy the ensemble model
$ go run ./examples-go/grpc_client.go predict -n yolov4 --version 1 -f sample-models/dog.jpg # make inference
```

### Create a your own model to run in Triton server

## Community support

For general help using VDP, you can use one of these channels:

- [GitHub](https://github.com/instill-ai/vdp) (bug reports, feature requests, project discussions and contributions)
- [Discord](https://discord.gg/sevxWsqpGh) (live discussion with the community and the Instill AI Team)

## Documentation

### API reference

### Build docker

You can build a development Docker image using:
```bash
$ docker build -t {tag} .
```

## License

See the [LICENSE](https://github.com/instill-ai/vdp/blob/main/LICENSE) file for licensing information.
