.DEFAULT_GOAL:=help

INSTILL_SERVICES := model-backend-migrate model-backend triton-conda-env
3RD_PARTY_SERVICES := triton-server database
ALL_SERVICES := ${INSTILL_SERVICES} ${3RD_PARTY_SERVICES}

#============================================================================

#============================================================================

all:			## Build and launch all services
	@docker inspect --type=image nvcr.io/nvidia/tritonserver:${TRITONSERVER_VERSION} >/dev/null 2>&1 || printf "\033[1;33mWARNING:\033[0m This may take a while due to the enormous size of the Triton server image, but the image pulling process should be just a one-time effort.\n" && sleep 5
	@docker-compose up -d ${ALL_SERVICES}	
.PHONY: all

logs:			## Tail all logs with -n 10
	@docker-compose logs --follow --tail=10
.PHONY: logs

pull:			## Pull all service images
	@docker inspect --type=image nvcr.io/nvidia/tritonserver:${TRITONSERVER_VERSION} >/dev/null 2>&1 || printf "\033[1;33mWARNING:\033[0m This may take a while due to the enormous size of the Triton server image, but the image pulling process should be just a one-time effort.\n" && sleep 5
	@docker-compose pull ${ALL_SERVICES}
.PHONY: pull

stop:			## Stop all components
	@docker-compose stop ${ALL_SERVICES}
.PHONY: stop

start:			## Start all stopped services
	@docker-compose start ${ALL_SERVICES}
.PHONY: start

restart:		## Restart all services
	@docker-compose restart ${ALL_SERVICES}
.PHONY: restart

rm:				## Remove all stopped service containers
	@docker-compose rm -f ${ALL_SERVICES}
.PHONY: rm

down:			## Stop all services and remove all service containers
	@docker-compose down
.PHONY: down

images:			## List all container images
	@docker-compose images ${ALL_SERVICES}
.PHONY: images

ps:				## List all service containers
	@docker-compose ps ${ALL_SERVICES}
.PHONY: ps

top:			## Display all running service processes
	@docker-compose top ${ALL_SERVICES}
.PHONY: top

prune:			## Remove all services containers and system prune everything
	@make down
	@docker system prune -f --volumes
.PHONY: prune

build:			## Build local docker image
	@DOCKER_BUILDKIT=1 docker build -t instill/model-backend:dev .
.PHONY: build

help:       	## Show this help
	@echo "\nMake Application using Docker-Compose files."
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m (default: help)\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
.PHONY: help

test: prepare-test integration-test cleanup-test			## Run integration test
.PHONY: test

prepare-test:
	@go version
	@go install go.k6.io/xk6/cmd/xk6@latest
	@xk6 build --with github.com/szkiba/xk6-jose@latest
.PHONY: prepare-test

cleanup-test:
	@rm k6
.PHONY: cleanup-test

integration-test:
	@TEST_FOLDER_ABS_PATH=${PWD} ./k6 run tests/integration-tests/model-backend-rest.js --no-usage-report
.PHONE: integration-test