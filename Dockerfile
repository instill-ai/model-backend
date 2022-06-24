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

FROM --platform=$BUILDPLATFORM alpine:3.16.0

ARG SERVICE_NAME

RUN apk update
RUN apk add git git-lfs

# Install python/pip
ENV PYTHONUNBUFFERED=1
RUN apk add --update --no-cache python3-dev && ln -sf python3 /usr/bin/python
RUN python3 -m ensurepip
RUN pip3 install --no-cache --upgrade pip setuptools
RUN apk add --no-cache libffi-dev build-base py3-pip python3-dev
RUN apk add --no-cache libgit2-dev py3-pygit2
RUN pip3 install dvc[gs]

WORKDIR /${SERVICE_NAME}

COPY --from=build /src/config ./config
COPY --from=build /src/release-please ./release-please
COPY --from=build /src/internal/db/migration ./internal/db/migration

COPY --from=build /${SERVICE_NAME}-migrate ./
COPY --from=build /${SERVICE_NAME}-init ./
COPY --from=build /${SERVICE_NAME} ./

# ArtiVC tool to work with cloud storage
COPY --from=build /src/third_party/ArtiVC/bin/avc /bin/avc
