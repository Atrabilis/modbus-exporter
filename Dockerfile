FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o modbus-exporter ./cmd/modbus-exporter

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /src/modbus-exporter /usr/local/bin/modbus-exporter
ENTRYPOINT ["/usr/local/bin/modbus-exporter"]

