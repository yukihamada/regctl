FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=$(cat VERSION 2>/dev/null || echo dev) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o regctl ./cmd/regctl

FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /build/regctl /usr/local/bin/regctl

EXPOSE 8080

ENTRYPOINT ["regctl"]
CMD ["server", "--port", "8080"]
