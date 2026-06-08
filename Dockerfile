# BUILD STAGE
FROM golang:1.26-trixie AS builder

# Install Git and CA Certificates (Debian uses apt)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/app

# Copy dependency files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy remaining source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/server

# FINAL STAGE
FROM scratch AS prod

# Copy CA Certificates from builder to enable HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /go/bin/server /go/bin/server

ENTRYPOINT ["/go/bin/server"]
