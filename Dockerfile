FROM golang:1.17.2 AS build

WORKDIR /go/src
COPY . /go/src

ENV CGO_ENABLED=0

RUN go get -d -v ./...

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend ./cmd/main
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend-migrate ./cmd/migration
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend-init ./cmd/init

FROM alpine:3.5 AS runtime
RUN apk update
RUN apk add git

WORKDIR /model-backend

COPY --from=build /model-backend ./
COPY --from=build /model-backend-migrate ./
COPY --from=build /model-backend-init ./
COPY --from=build /go/src/config ./config
COPY --from=build /go/src/internal/db/migration ./internal/db/migration

ENTRYPOINT ["./model-backend"]
