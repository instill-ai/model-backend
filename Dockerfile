FROM golang:1.17.2 AS build

WORKDIR /go/src

ENV CGO_ENABLED=0

# Copy go.mod and go.sum
COPY go.* .
RUN go mod download

# Compile source codes and cache it

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend ./cmd
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /model-backend-migrate ./migrate

FROM gcr.io/distroless/base AS runtime

WORKDIR /model-backend

COPY --from=build /model-backend ./model-backend
COPY --from=build /go/src/configs ./configs/
COPY --from=build /go/src/internal/db/migrations ./internal/db/migrations/
COPY --from=build /model-backend-migrate ./

EXPOSE 8080/tcp
ENTRYPOINT ["./model-backend"]
