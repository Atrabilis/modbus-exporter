FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -o modbus-exporter ./cmd/modbus-exporter

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /src/modbus-exporter /usr/local/bin/modbus-exporter
ENTRYPOINT ["/usr/local/bin/modbus-exporter"]
