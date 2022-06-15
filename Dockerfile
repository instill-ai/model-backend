FROM golang:1.18.2 AS build

WORKDIR /go/src
COPY . /go/src

ENV CGO_ENABLED=0

RUN go get -d -v ./...

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend ./cmd/main
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend-migrate ./cmd/migration
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend-init ./cmd/init

WORKDIR /go/src/third_party/ArtiVC 
ENV CGO_ENABLED=0

RUN go get -d -v ./...
RUN make build

FROM alpine:3.16.0 AS runtime
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

WORKDIR /model-backend

COPY --from=build /model-backend ./
COPY --from=build /model-backend-migrate ./
COPY --from=build /model-backend-init ./

COPY --from=build /go/src/config ./config
COPY --from=build /go/src/release-please ./release-please
COPY --from=build /go/src/internal/db/migration ./internal/db/migration

# ArtiVC tool to work with cloud storage
COPY --from=build /go/src/third_party/ArtiVC/bin/avc /bin/avc


ENTRYPOINT ["./model-backend"]
