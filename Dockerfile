# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy everything else
COPY . .

# Generate protobuf stubs
RUN chmod +x scripts/generate_proto.sh && \
    scripts/generate_proto.sh

# Build the binary
RUN go build -o /usr/local/bin/myservice ./cmd/server

# Final minimal image
FROM alpine:3.18
RUN apk add --no-cache ca-certificates

COPY --from=builder /usr/local/bin/myservice /usr/local/bin/myservice
COPY internal/config/config.yaml /etc/myservice/config.yaml

EXPOSE 50051

ENTRYPOINT ["/usr/local/bin/myservice"]
