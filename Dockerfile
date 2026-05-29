FROM golang:1.15-alpine AS builder
# Install Git
RUN apk update && apk add --no-cache git
# Copy In Source Code
WORKDIR /go/src/app
COPY . .
# Install Dependencies
RUN go get -d -v
# Build
RUN go get -d -v \
    && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -ldflags="-w -s" -o /go/bin/server
# SCRATCH IMAGE
FROM scratch AS prod
COPY --from=builder /go/bin/server /go/bin/server
ENTRYPOINT ["/go/bin/server"]
