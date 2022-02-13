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
$ go run ./examples-go/grpc_client.go predict -n ensemble -t 1 -f sample-models/dog.jpg # -t 0: classification model and 1: object detection model; yolov4 is detection model
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
