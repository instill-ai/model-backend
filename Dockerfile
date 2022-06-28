FROM --platform=$BUILDPLATFORM golang:1.18.2 AS build

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

WORKDIR /src/third_party

RUN git clone https://github.com/InfuseAI/ArtiVC && cd ArtiVC && git checkout tags/v0.9.0 && go get -d -v ./... && make build

FROM --platform=$BUILDPLATFORM ubuntu:20.04

RUN apt update && \
    apt install -y bash \
                   build-essential \
                   git git-lfs\
                   python3 \
                   python3-pip && \
    rm -rf /var/lib/apt/lists
RUN python3 -m pip install --no-cache-dir transformers pillow torch onnxruntime dvc[gs]

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=build /src/config ./config
COPY --from=build /src/release-please ./release-please
COPY --from=build /src/internal/db/migration ./internal/db/migration

COPY --from=build /${SERVICE_NAME}-migrate ./
COPY --from=build /${SERVICE_NAME}-init ./
COPY --from=build /${SERVICE_NAME} ./

# ArtiVC tool to work with cloud storage
COPY --from=build /src/third_party/ArtiVC/bin/avc /bin/avc

