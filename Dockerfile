ARG GOLANG_VERSION
ARG UBUNTU_VERSION

FROM --platform=$BUILDPLATFORM golang:${GOLANG_VERSION} AS build

ARG SERVICE_NAME

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ENV CGO_ENABLED=0

ARG TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME} ./cmd/main
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-migrate ./cmd/migration
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-init ./cmd/init
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-worker ./cmd/worker

WORKDIR /src/third_party

RUN git clone https://github.com/InfuseAI/ArtiVC && cd ArtiVC && git checkout tags/v0.9.0 && go get -d -v ./... && make build

FROM --platform=$BUILDPLATFORM ubuntu:${UBUNTU_VERSION}

RUN apt update && \
    apt install -y bash \
    build-essential \
    python3 python3-setuptools python3-pip git git-lfs && \
    rm -rf /var/lib/apt/lists
RUN pip3 install --upgrade pip setuptools wheel
RUN pip3 install --no-cache-dir transformers==4.21.0 pillow torch==1.12.1 torchvision==0.13.1 onnxruntime==1.11.1 dvc[gs]==2.34.2

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=docker:dind /usr/local/bin/docker /usr/local/bin/

COPY --from=build /src/config ./config
COPY --from=build /src/assets ./assets
COPY --from=build /src/release-please ./release-please
COPY --from=build /src/internal/db/migration ./internal/db/migration

COPY --from=build /${SERVICE_NAME}-migrate ./
COPY --from=build /${SERVICE_NAME}-init ./
COPY --from=build /${SERVICE_NAME} ./
COPY --from=build /${SERVICE_NAME}-worker ./

# ArtiVC tool to work with cloud storage
COPY --from=build /src/third_party/ArtiVC/bin/avc /bin/avc
