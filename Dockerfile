FROM golang:1.17.2 AS build
WORKDIR /go/src

COPY go.mod .
COPY go.sum .

ENV CGO_ENABLED=0
RUN go mod download

COPY . /go/src

RUN --mount=type=cache,target=/root/.cache/go-build go build -a -o /model-backend ./cmd
RUN --mount=type=cache,target=/root/.cache/go-build go build -a -o /model-backend-migrate ./migrate


FROM gcr.io/distroless/base AS runtime

ENV GIN_MODE=release
WORKDIR /model-backend

COPY --from=build /model-backend ./model-backend
COPY --from=build /go/src/configs ./configs/
COPY --from=build /go/src/internal/db/migrations ./internal/db/migrations/
COPY --from=build /model-backend-migrate ./

EXPOSE 8080/tcp
ENTRYPOINT ["./model-backend"]
