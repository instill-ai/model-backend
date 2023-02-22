ARG GOLANG_VERSION
ARG UBUNTU_VERSION

FROM --platform=$BUILDPLATFORM golang:${GOLANG_VERSION} AS build

ARG SERVICE_NAME

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME} ./cmd/main
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-migrate ./cmd/migration
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-init ./cmd/init
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-worker ./cmd/worker

# ArtiVC to work with cloud storage
ARG TARGETOS TARGETARCH ARTIVC_VERSION
ADD https://github.com/InfuseAI/ArtiVC/releases/download/v${ARTIVC_VERSION}/ArtiVC-v${ARTIVC_VERSION}-${TARGETOS}-${TARGETARCH}.tar.gz ArtiVC-v${ARTIVC_VERSION}-${TARGETOS}-${TARGETARCH}.tar.gz
RUN tar -xf ArtiVC-v${ARTIVC_VERSION}-${TARGETOS}-${TARGETARCH}.tar.gz -C /usr/local/bin

# Mounting points
RUN mkdir /etc/vdp
RUN mkdir /vdp
RUN mkdir /model-repository

FROM --platform=$BUILDPLATFORM ubuntu:${UBUNTU_VERSION}

RUN apt update && apt install -y \
    bash \
    build-essential \
    python3 \
    python3-setuptools \
    python3-pip \
    git \
    git-lfs \
    curl \
    && rm -rf /var/lib/apt/lists/*
RUN pip3 install --upgrade pip setuptools wheel
RUN pip3 install --no-cache-dir transformers==4.21.0 pillow torch==1.12.1 torchvision==0.13.1 onnxruntime==1.11.1 dvc[gs]==2.34.2

# Need permission of /tmp folder for internal process such as store temporary files
RUN chown -R nobody:nogroup /tmp

USER nobody:nogroup

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=docker:dind-rootless --chown=nobody:nogroup /usr/local/bin/docker /usr/local/bin

COPY --from=build --chown=nobody:nogroup /src/config ./config
COPY --from=build --chown=nobody:nogroup /src/assets ./assets
COPY --from=build --chown=nobody:nogroup /src/release-please ./release-please
COPY --from=build --chown=nobody:nogroup /src/internal/db/migration ./internal/db/migration

COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-migrate ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-init ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME} ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-worker ./
COPY --from=build --chown=nobody:nogroup /usr/local/bin/avc /usr/local/bin/avc

COPY --from=build --chown=nobody:nogroup /etc/vdp /etc/vdp
COPY --from=build --chown=nobody:nogroup /vdp /vdp
COPY --from=build --chown=nobody:nogroup /model-repository /model-repository