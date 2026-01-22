# modbus-exporter

Prometheus exporter for Modbus devices.

`modbus-exporter` polls Modbus TCP devices and exposes metrics over HTTP for Prometheus scraping.

---

## Features

- Modbus TCP polling
- YAML-based configuration
- Prometheus `/metrics` endpoint
- `/health` endpoint for liveness
- Docker-ready

---

## Quick start

Build locally:

```bash
go build -o modbus-exporter ./cmd/modbus-exporter
```

Run:

```
./modbus-exporter --config config.yml
```

Or with docker with a docker-compose like the one below

```
sudo docker compose up -d
```

Metrics:

- http://localhost:9105/metrics

---

## Configuration

Configuration is provided via a YAML file.

Example:

```yaml
server:
  listen: ":9105"

poll_interval: 10s

modbus:
  timeout: 3s
  retries: 2

devices:
  - name: inverter_1
    endpoint: tcp://192.168.1.50:502
    unit_id: 1

    registers:
      - name: active_power
        address: 30001
        type: U32
        scale: 0.1
        help: "Active power in kW"
```
A more complete yaml can be found at the internal/config folder of the repo

---

## HTTP endpoints

- `/metrics` — Prometheus metrics
- `/health` — liveness check

---

## Docker

Build image:

```
docker build -t atrabilis/modbus-exporter:v0.1.0 .
```

Run:

```
docker run \
  -p 9105:9105 \
  -v $(pwd)/config.yml:/etc/modbus-exporter/config.yml:ro \
  atrabilis/modbus-exporter:v0.1.0 \
  --config /etc/modbus-exporter/config.yml
```

---

## Docker Compose

Example docker-compose.yml:

```yaml
services:
  modbus-exporter:
    image: atrabilis/modbus-exporter:v0.1.0
    container_name: modbus-exporter

    restart: unless-stopped

    ports:
      - "9105:9105"

    volumes:
      - ./your-config.yml:/your-config.yml:ro

    command:
      - "--config"
      - "/your-config.yml"
      - "--debug"

    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:9105/health"]
      interval: 30s
      timeout: 5s
      retries: 3

```

your-config.yml has to be in the same folder as the compose.

---

## Versioning

This project follows Semantic Versioning.

- 0.x.y — unstable API (metrics/config may change)
- 1.0.0 — stable metrics and configuration

---

## Production notes

- Use versioned Docker tags in production.
- Avoid `latest` in critical systems.
- Restart container after changing configuration.

---

## Docker images

Images are published as:

atrabilis/modbus-exporter:<tag>

Tags:

- v0.1.0 → release
- latest → latest stable
- test → development only

---

## Upgrade procedure

1. Update docker-compose image tag.
2. Pull new image:

```
docker compose pull
```

3. Restart:

```
docker compose up -d
```

---

## Rollback

Revert the image tag to the previous version and restart the container.

---

## License

MIT
