.DEFAULT_GOAL:=help

DB := pg_sql
TRITON := triton_server

#============================================================================

# load environment variables for local development
include .env
export

ifndef GOPATH
GOPATH := $(shell go env GOPATH)
endif

GOBIN := $(if $(shell go env GOBIN),$(shell go env GOBIN),$(GOPATH)/bin)
PATH := $(GOBIN):$(PATH)

DEV_DB_MIGRATION_BINARY := $(shell mktemp -d)/model-backend-migrate
DEV_DB_INIT_BINARY := $(shell mktemp -d)/model-backend-init

K6BIN := $(if $(shell command -v k6 2> /dev/null),k6,$(shell mktemp -d)/k6)

TRITONSERVER_IMAGE_TAG := $(if $(filter arm64,$(shell uname -m)),instill/tritonserver:22.05-py3-cpu-arm64,nvcr.io/nvidia/tritonserver:22.05-py3)
TRITONCONDAENV_IMAGE_TAG := $(if $(filter arm64,$(shell uname -m)),instill/triton-conda-env:0.2.2-alpha-cpu-arm64,instill/triton-conda-env:0.2.2-alpha-cpu)
REDIS_IMAGE_TAG := $(if $(filter arm64,$(shell uname -m)),arm64v8/redis:7.0-alpine,amd64/redis:7.0-alpine)
#============================================================================

.PHONY: all
all:							## Build and launch all services
	@docker inspect --type=image ${TRITONSERVER_IMAGE_TAG} >/dev/null 2>&1 || printf "\033[1;33mWARNING:\033[0m This may take a while due to the enormous size of the Triton server image, but the image pulling process should be just a one-time effort.\n" && sleep 5
	docker-compose up -d

.PHONY: dev
dev:							## Lunch only dependant services for local development
	@docker-compose up -d ${DB} ${TRITON}
	while [ "$$(docker inspect --format '{{ .State.Health.Status }}' pg-sql)" != "healthy" ]; do echo "Check if db is ready..." && sleep 1; done
	go build -o ${DEV_DB_MIGRATION_BINARY} ./cmd/migration && ${DEV_DB_MIGRATION_BINARY} && rm -rf $(dirname ${DEV_DB_MIGRATION_BINARY})
	go build -o ${DEV_DB_INIT_BINARY} ./cmd/init && ${DEV_DB_INIT_BINARY} && rm -rf $(dirname ${DEV_DB_INIT_BINARY})

.PHONY: logs
logs:							## Tail all logs with -n 10
	@docker-compose logs --follow --tail=10

.PHONY: pull
pull:							## Pull all service images
	@docker inspect --type=image ${TRITONSERVER_IMAGE_TAG} >/dev/null 2>&1 || printf "\033[1;33mWARNING:\033[0m This may take a while due to the enormous size of the Triton server image, but the image pulling process should be just a one-time effort.\n" && sleep 5
	docker-compose pull

.PHONY: stop
stop:							## Stop all components
	@docker-compose stop

.PHONY: start
start:							## Start all stopped services
	@docker-compose start

.PHONY: restart
restart:						## Restart all services
	@docker-compose restart

.PHONY: rm
rm:								## Remove all stopped service containers
	@docker-compose rm -f

.PHONY: down
down:							## Stop all services and remove all service containers and volumes
	@docker-compose down -v

.PHONY: images
images:							## List all container images
	@docker-compose images

.PHONY: ps
ps:								## List all service containers
	@docker-compose ps

.PHONY: top
top:							## Display all running service processes
	@docker-compose top

.PHONY: build
build:							## Build local docker image
	docker build -t instill/model-backend:dev .

.PHONY: go-gen
go-gen:       					## Generate codes from "//go:generate" comments in the codebase (e.g., mockgen for unit tests)
	go generate ./...

.PHONY: unit-test
unit-test:       				## Run unit test
	@go test -v -race -coverpkg=./... -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out
	@rm coverage.out

.PHONY: integration-test
integration-test:				## Run integration test
	@if [ ${K6BIN} != "k6" ]; then\
		echo "Install k6 binary at ${K6BIN}";\
		go version;\
		go install go.k6.io/xk6/cmd/xk6@latest;\
		xk6 build --with github.com/szkiba/xk6-jose@latest --output ${K6BIN};\
	fi
	@TEST_FOLDER_ABS_PATH=${PWD} ${K6BIN} run integration-test/rest.js --no-usage-report
	@TEST_FOLDER_ABS_PATH=${PWD} ${K6BIN} run integration-test/grpc.js --no-usage-report
	@if [ ${K6BIN} != "k6" ]; then rm -rf $(dirname ${K6BIN}); fi

.PHONY: help
help:       					## Show this help
	@echo "\nMake application using Docker-Compose files."
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m (default: help)\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
