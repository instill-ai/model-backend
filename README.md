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
$ go run ./examples-go/grpc_client.go upload -f yolov4-onnx-cpu.zip # upload model file
$ go run ./examples-go/grpc_client.go load -n ensemble
$ go run  ./examples-go/grpc_client.go prdict -n ensemble -f https://artifacts.instill.tech/dog.jpg
```

Note: check the server running via `curl`:

```bash
$ curl http://localhost:8080/__health
{"status":"ok"}%

$ curl http://localhost:8080/v1/helloworld
{"error":"You've not implemented this API yet"}%
```

## In the nut shell

Before building the scaffold:
```
.
├── Dockerfile
├── README.md
├── scripts
│   ├── build.sh
│   └── run-redoc.sh
├── templates
│   └── go-gin
│       ├── openapi-generator-config.yaml  // openapi-generator configuration file
│       ├── openapi-generator-templates    // all openapi-generator template files
│       │   ├── controller-api-handler.mustache
│       │   ├── controller-api.mustache
│       │   ├── init.mustache
│       │   ├── main.mustache
│       │   ├── middlewares.mustache
│       │   ├── model.mustache
│       │   └── routers.mustache
│       └── openapi.yaml.tmpl              // openapi.yaml template file
└── version.txt
```

After running `scripts/build.sh -c` (remember to replace the `BACKEND_NAME` variable with your backend service name in that script):

```
.
├── Dockerfile
├── README.md
├── api
│   └── openapi.yaml
├── cmd
│   └── openapi
│       └── main.go
├── go.mod
├── go.sum
├── openapi
├── pkg
│   ├── controllers
│   │   └── helloworld.go
│   ├── handlers
│   │   ├── helloworld.go  // !! Start your handler implementation here !!
│   │   └── init.go
│   ├── models
│   │   ├── error.go
│   │   ├── error_all_of.go
│   │   ├── hello_world.go
│   │   ├── inline_response_200.go
│   │   └── request_info.go
│   └── routers
│       ├── middlewares.go
│       └── routers.go
├── scripts
│   ├── build.sh
│   └── run-redoc.sh
├── templates
│   └── go-gin
│       ├── openapi-generator-config.yaml
│       ├── openapi-generator-templates
│       │   ├── controller-api-handler.mustache
│       │   ├── controller-api.mustache
│       │   ├── init.mustache
│       │   ├── main.mustache
│       │   ├── middlewares.mustache
│       │   ├── model.mustache
│       │   └── routers.mustache
│       └── openapi.yaml.tmpl
└── version.txt
```

You can now focus on only the handler implementation under `pkg/handlers` folder, and check-in the files:
```
$ git add api cmd pkg go.mod go.sum
```

## Logging with [Logrus](https://github.com/sirupsen/logrus)

The loggers are maintained in [middlewares.mustache](templates/go-gin/openapi-generator-templates/middlewares.mustache). We log as JSON instead of the default ASCII formatter for Gin logger and debug logger.

### Gin logger
The Gin logger logs every request including
```JSON
{
  "Jwt-Client-Id":"",
  "Jwt-Scope":"",
  "Jwt-Sub":"",
  "Request-Id":"123",
  "backend":"test-backend",
  "clientIP":"::1",
  "dataLength":47,
  "latencyMs":936,
  "level":"error",
  "method":"GET",
  "msg":"{\"error\":\"You've not implemented this API yet\"}",
  "path":"/v1/pets/meow",
  "referer":"",
  "statusCode":500,
  "time":"2021/10/13 - 01:16:35.1001Z","userAgent":"PostmanRuntime/7.28.3"
}
```

### Debug logger
The debug logger is for debugging purpose. Here is an example showing how to log anything about a request in a handler
```Go
// GetPetById - Find pet by ID
func (r *PetsControllerImp) GetPetById(c *gin.Context) {
	l := c.MustGet("Logger").(*zap.Logger)
	l.WithField("customField", "log any field required").Info("log me!")

	c.JSON(http.StatusInternalServerError, gin.H{"error": "You've not implemented this API yet"})
}
```

```JSON
{
  "Request-Id": "123",
  "backend": "test-backend",
  "customField": "log any field required",
  "level": "info",
  "msg": "log me!",
  "time": "2021/10/13 - 01:16:35.1001Z"
}
```

## How to contribute

### Commit changes and submit PRs

- Make PRs as small as possible. Each PR should only solve one problem. Nobody wants to review 5000+ lines of codes.
- Never commit auto-generated code. The auto-generated code will be included in the production image automatically by the CD workflow.


## CI/CD

1. Create a feature branch to implement new features
2. Create a PR for that feature branch and merge it to `main`
3. Merging the PR will
   1. make the `release-please` bot either create a new release PR or update the existing release PR
   2. build and push the Docker images with tag `<current-version>_<short-sha>`.
4. Once all PRs for a version are ready, we can merge the release PR, which will
   1. make the `release-please` bot create a tag and publish a release
   2. build and push the Docker image with tag `<current-version>`, e.g. `<replace-with-your-backend-name>:1.0.0`
   3. remove the temporary images tagged with a short sha.
   4. upload the API doc to q Cloudflare KV store.
   5. create an additional git tag for the API public version if `version-public.txt` was changed in step 2

### Push
With [Release Please Action](https://github.com/google-github-actions/release-please-action), we maintain two types of version for the container images, (almost) following [SemVer 2.0](https://semver.org):
1. Version core for release `push`: `<versioncore>`;
2. Build metadata version for non-release `push`: `<versioncore>_<buildmetadata>`

Unlike what is specified in [SemVer 2.0](https://semver.org), we use `_` instead of `+` to separate the version core and the build metadata because [Docker tags don't support `+`](https://github.com/opencontainers/distribution-spec/issues/154).

### Pull request
Each git `push` to a feature branch in a pull request (PR) session will trigger a container build tagged by its build metadata version and pushed to GAR image repository.

### Image purge
Two cases:
1. Images with a build metadata version will be purged every time after a PR is merged (pushed) into `main` branch. The image with the latest build metadata version (latest commit) will be kept.

2. All images with a build metadata version will be purged right after a new release is issued.
