FROM golang:1.17.2 AS build
WORKDIR /go/src

COPY go.mod .
COPY go.sum .

ENV CGO_ENABLED=0
RUN go mod download

COPY . /go/src

RUN --mount=type=cache,target=/root/.cache/go-build go build -a -o /openapi ./cmd
RUN --mount=type=cache,target=/root/.cache/go-build go build -a -o /migrate ./migrate

# ! DO NOT USE THIS SELF-SIGNED CERTIFICATE ON PRODUCTION
# Generate self-signed SSL/TLS certificate
WORKDIR /model-backend/ssl
# Create a private key ca.key
RUN openssl genrsa -out ca.key 2048
# Create a self-signed root Certificate Authority (root-CA) certificate ca.crt
RUN openssl req -new -x509 -days 365 -key ca.key -subj "/C=UK/ST=London/L=London/O=Instill AI/CN=localhost" -out ca.crt
# Create a Certificate Signing Request (CSR) tls.csr and its corresponding private key
RUN openssl req -newkey rsa:2048 -nodes -keyout tls.key -subj "/C=UK/ST=London/L=London/O=Instill AI/CN=localhost" -out tls.csr
# Convert a CSR into a self-signed certificate using extensions for a CA
RUN echo subjectAltName=DNS:localhost > extfile.cnf
RUN openssl x509 -req -extfile extfile.cnf -days 365 -in tls.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out tls.crt
RUN rm extfile.cnf

# Install the Root CA certificates
RUN cp ca.crt /etc/ssl/certs/ca.crt && update-ca-certificates

#
#
# Prod image
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4 AS runtime
# FROM gcr.io/distroless/base AS runtime

WORKDIR /model-backend

ENV GIN_MODE=release

COPY --from=build /openapi ./
COPY --from=build /go/src/configs ./configs/
COPY --from=build /go/src/internal/db/migrations ./internal/db/migrations/
COPY --from=build /migrate ./

COPY --from=build /model-backend ssl
COPY --from=build /model-backend /ssl/certs/ca.crt

ENTRYPOINT ["./openapi"]
