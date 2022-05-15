FROM golang:1.17.2 AS build

WORKDIR /go/src
COPY . /go/src

RUN go get -d -v ./...

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend ./cmd/main
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend-migrate ./cmd/migration
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend-init ./cmd/init

FROM gcr.io/distroless/base AS runtime

WORKDIR /model-backend

COPY --from=build /model-backend ./
COPY --from=build /model-backend-migrate ./
COPY --from=build /model-backend-init ./
COPY --from=build /go/src/configs ./configs
COPY --from=build /go/src/internal/db/migration ./internal/db/migration

ENTRYPOINT ["./model-backend"]
