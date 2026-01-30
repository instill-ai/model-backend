.DEFAULT_GOAL:=help

#============================================================================

# Load environment variables for local development
include .env
export

# Integration test configuration
# - From host: make integration-test (uses localhost:8080)
# - In container: make integration-test API_GATEWAY_URL=api-gateway:8080 DB_HOST=pg_sql
API_GATEWAY_PROTOCOL ?= http
API_GATEWAY_URL ?= localhost:8080
DB_HOST ?= localhost

#============================================================================

.PHONY: dev
dev:							## Run dev container
	@docker compose ls -q | grep -q "instill-core" && true || \
		(echo "Error: Run \"make latest PROFILE=exclude-mode\" in model repository (https://github.com/instill-ai/instill-core) in your local machine first and  run \"docker rm -f ${SERVICE_NAME} ${SERVICE_NAME}-worker\"" && exit 1)
	@docker inspect --type container ${SERVICE_NAME} >/dev/null 2>&1 && echo "A container named ${SERVICE_NAME} is already running." || \
		echo "Run dev container ${SERVICE_NAME}. To stop it, run \"make stop\"."
	@docker run -d --rm \
		-v $(PWD):/${SERVICE_NAME} \
		-p ${SERVICE_PORT}:${SERVICE_PORT} \
		-p ${PRIVATE_SERVICE_PORT}:${PRIVATE_SERVICE_PORT} \
		--network instill-network \
		--name ${SERVICE_NAME} \
		instill/${SERVICE_NAME}:dev

.PHONY: latest
latest: ## Run latest container
	@docker compose ls -q | grep -q "instill-core" && true || \
		(echo "Error: Run \"make latest\" in instill-core repository (https://github.com/instill-ai/instill-core) in your local machine first and run \"docker rm -f ${SERVICE_NAME} ${SERVICE_NAME}-worker\"" && exit 1)
	@docker inspect --type container ${SERVICE_NAME} >/dev/null 2>&1 && echo "A container named ${SERVICE_NAME} is already running." || \
		echo "Run latest container ${SERVICE_NAME} and ${SERVICE_NAME}-worker. To stop it, run \"make stop\"."
	@docker run --network=instill-network \
		--name ${SERVICE_NAME} \
		-d instill/${SERVICE_NAME}:latest \
		/bin/sh -c "\
		./${SERVICE_NAME}-migrate && \
		./${SERVICE_NAME}-init && \
		./${SERVICE_NAME} \
		"
	@docker run --network=instill-network \
		--name ${SERVICE_NAME}-worker \
		-d instill/${SERVICE_NAME}:latest ./${SERVICE_NAME}-worker

.PHONY: rm
rm: ## Remove all running containers
	@docker rm -f ${SERVICE_NAME} ${SERVICE_NAME}-worker >/dev/null 2>&1

.PHONY: logs
logs:							## Tail container logs with -n 10
	@docker logs ${SERVICE_NAME} --follow --tail=10

.PHONY: stop
stop:							## Stop container
	@docker stop -t 1 ${SERVICE_NAME}

.PHONY: top
top:							## Display all running service processes
	@docker top ${SERVICE_NAME}

.PHONY: build-dev
build-dev: ## Build dev docker image
	@docker build \
		--build-arg K6_VERSION=${K6_VERSION} \
		--build-arg SERVICE_NAME=${SERVICE_NAME} \
		--build-arg SERVICE_VERSION=dev \
		-f Dockerfile.dev -t instill/${SERVICE_NAME}:dev .

.PHONY: build-latest
build-latest: ## Build latest docker image
	@docker build \
		--build-arg SERVICE_NAME=${SERVICE_NAME} \
		--build-arg SERVICE_VERSION=dev \
		-t instill/${SERVICE_NAME}:latest .

.PHONY: go-gen
go-gen:       					## Generate codes
	go generate ./...

.PHONY: unit-test
unit-test:       				## Run unit test
	@go test -v -race -coverpkg=./... -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out
	@rm coverage.out

.PHONY: integration-test
integration-test:				## Run integration test (CE: Basic Auth only)
	@echo "✓ Running tests via API Gateway: ${API_GATEWAY_URL}"
	@echo "  DB_HOST: ${DB_HOST}"
	@rm -f /tmp/model-integration-test.log
	@TEST_FOLDER_ABS_PATH=${PWD} k6 run --address="" \
		-e API_GATEWAY_PROTOCOL=${API_GATEWAY_PROTOCOL} \
		-e API_GATEWAY_URL=${API_GATEWAY_URL} \
		-e DB_HOST=${DB_HOST} \
		integration-test/grpc.js --no-usage-report 2>&1 | tee -a /tmp/model-integration-test.log
	@TEST_FOLDER_ABS_PATH=${PWD} k6 run --address="" \
		-e API_GATEWAY_PROTOCOL=${API_GATEWAY_PROTOCOL} \
		-e API_GATEWAY_URL=${API_GATEWAY_URL} \
		-e DB_HOST=${DB_HOST} \
		integration-test/rest.js --no-usage-report 2>&1 | tee -a /tmp/model-integration-test.log
	@TEST_FOLDER_ABS_PATH=${PWD} k6 run --address="" \
		-e API_GATEWAY_PROTOCOL=${API_GATEWAY_PROTOCOL} \
		-e API_GATEWAY_URL=${API_GATEWAY_URL} \
		-e DB_HOST=${DB_HOST} \
		integration-test/rest-with-basic-auth.js --no-usage-report 2>&1 | tee -a /tmp/model-integration-test.log

.PHONY: help
help:       	 				## Show this help
	@echo "\nMakefile for local development"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m (default: help)\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
