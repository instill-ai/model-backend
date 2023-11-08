ARG GOLANG_VERSION
ARG UBUNTU_VERSION

FROM --platform=$BUILDPLATFORM golang:${GOLANG_VERSION} AS build

ARG SERVICE_NAME

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /${SERVICE_NAME} ./cmd/main
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-migrate ./cmd/migration
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-init ./cmd/init
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-worker ./cmd/worker
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-init-model ./cmd/model

# ArtiVC to work with cloud storage
ARG TARGETOS TARGETARCH ARTIVC_VERSION
ADD https://github.com/InfuseAI/ArtiVC/releases/download/v${ARTIVC_VERSION}/ArtiVC-v${ARTIVC_VERSION}-${TARGETOS}-${TARGETARCH}.tar.gz ArtiVC-v${ARTIVC_VERSION}-${TARGETOS}-${TARGETARCH}.tar.gz
RUN tar -xf ArtiVC-v${ARTIVC_VERSION}-${TARGETOS}-${TARGETARCH}.tar.gz -C /usr/local/bin

# Mounting points
RUN mkdir /etc/vdp
RUN mkdir /vdp
RUN mkdir /model-repository
RUN mkdir /.cache

FROM --platform=$BUILDPLATFORM golang:${GOLANG_VERSION}

ENV DEBIAN_FRONTEND noninteractive

# tools to work with versatile model import
RUN apt-get update && apt-get install -y \
    python3 \
    python3-setuptools \
    python3-pip \
    python3-venv \
    git \
    git-lfs \
    curl \
    && rm -rf /var/lib/apt/lists/*

ENV VENV=/opt/venv
RUN python3 -m venv $VENV
ENV PATH="$VENV/bin:$PATH"

RUN pip install jsonschema pyyaml
RUN pip install --upgrade pip setuptools wheel
RUN pip install --no-cache-dir opencv-contrib-python-headless transformers pillow torch torchvision onnxruntime dvc[gs]==2.34.2
RUN pip install --no-cache-dir ray[serve] scikit-image
RUN pip install --no-cache-dir instill-sdk==0.3.1

# Need permission of /tmp folder for internal process such as store temporary files.
RUN chown -R nobody:nogroup /tmp
# Need permission of /nonexistent folder for HuggingFace internal process.
RUN mkdir /nonexistent > /dev/null && chown -R nobody:nogroup /nonexistent

USER nobody:nogroup

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=docker:dind-rootless --chown=nobody:nogroup /usr/local/bin/docker /usr/local/bin

COPY --from=build --chown=nobody:nogroup /src/config ./config
COPY --from=build --chown=nobody:nogroup /src/assets ./assets
COPY --from=build --chown=nobody:nogroup /src/release-please ./release-please
COPY --from=build --chown=nobody:nogroup /src/pkg/db/migration ./pkg/db/migration

COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-migrate ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-init ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME} ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-worker ./
COPY --from=build --chown=nobody:nogroup /${SERVICE_NAME}-init-model ./
COPY --from=build --chown=nobody:nogroup /usr/local/bin/avc /usr/local/bin/avc

COPY --from=build --chown=nobody:nogroup /etc/vdp /etc/vdp
COPY --from=build --chown=nobody:nogroup /vdp /vdp
COPY --from=build --chown=nobody:nogroup /model-repository /model-repository
COPY --from=build --chown=nobody:nogroup /.cache /.cache
