# modbus-exporter

Prometheus exporter for Modbus TCP devices.

`modbus-exporter` polls Modbus devices and exposes metrics over HTTP for Prometheus scraping.

---

## Requirements

Before running:

- Docker (recommended) OR Go >= 1.25
- Network connectivity to Modbus TCP devices
- IP address, port and unit ID of the target device

---

## Supported Platforms

This image is published as a multi-architecture Docker manifest since v0.1.1 and can run on:
- linux/amd64 
- linux/arm64 
- linux/arm/v7 
- linux/arm/v6 
- linux/386 

## Features

- Modbus TCP polling
- YAML-based configuration
- Prometheus `/metrics` endpoint
- `/health` endpoint for liveness
- Structured labels for devices and registers
- Docker-ready

---

## Quick start (Docker)

1) Create a configuration file:

```bash
nano config.yml
```

2) Run container:

```bash
docker run \
  -p 9105:9105 \
  -v $(pwd)/config.yml:/etc/modbus-exporter/config.yml:ro \
  atrabilis/modbus-exporter:v0.1.0 \
  --config /etc/modbus-exporter/config.yml
```

3) Verify:

```bash
curl http://localhost:9105/health
curl http://localhost:9105/metrics
```

---

## Example metrics output

With a device configured as:

- device: ion_7650
- unit ID: 1
- register: 40162

You should see output similar to:

```
modbus_value{device="ion_7650",slave="1",register="40162",name="freq_mean",unit="Hz",ip_address="192.168.100.8"} 50.000000
modbus_sample_age_seconds{device="ion_7650",slave="1",register="40162",ip_address="192.168.100.8"} 11.645795
```

Meaning:

- modbus_value → last sampled value
- modbus_sample_age_seconds → seconds since last successful poll
- labels identify device, register, unit and IP

---

## Quick start (binary)

Build locally:

```bash
go build -o modbus-exporter ./cmd/modbus-exporter
```

Run:

```bash
./modbus-exporter --config config.yml --debug
```

Metrics:

- http://localhost:9105/metrics

---

## Configuration

Configuration is provided via a YAML file.

Example:

```yaml
poll_interval: 60s

devices:
  - name: "ion_7650"
    protocol: "modbus-tcp"
    address: "192.168.100.8"
    port: 502
    timeout: 1s

    slaves:
      - name: "ion_7650"
        slave_id: 1
        offset: 40001
        modbus_registers:
          - register: 40162
            function_code: 3
            name: "freq_mean"
            description: "Mean frequency"
            words: 1
            datatype: "U16"
            unit: "Hz"
            gain: 0.1
            flags:
              module_number: 16
              module_label: "Amp/freq/unbal"
```

More detailed example:

```yaml
# This configuration defines how Modbus devices are polled
# and how their registers are exposed as Prometheus metrics.
#
# The exporter assumes the configuration is correct and does not
# attempt to infer offsets, datatypes, or scaling automatically.


# Supported datatypes (case-sensitive):
#
# Integer:
#   - U8       : Unsigned 8-bit (low byte)
#   - U16      : Unsigned 16-bit, big-endian
#   - S16      : Signed 16-bit, big-endian
#   - U32      : Unsigned 32-bit, big-endian
#   - S32      : Signed 32-bit, big-endian
#   - U32LE    : Unsigned 32-bit, little-endian by word (CDAB)
#   - S32LE    : Signed 32-bit, little-endian by word
#   - U64BE    : Unsigned 64-bit, big-endian
#   - S64BE    : Signed 64-bit, big-endian
#
# Float:
#   - F32BE    : Float32, big-endian (ABCD)
#   - F32LE    : Float32, little-endian by word (CDAB)
#   - F32CDAB  : Float32, word-swapped
#   - F32BADC  : Float32, byte-swapped
#   - F64BE    : Float64, big-endian
#
# String:
#   - UTF8     : C-style UTF-8 string (NUL-terminated / padded)
#
# Notes:
# - Modbus registers are 16-bit words.
# - "words" indicates how many 16-bit registers are read.
# - Offset is applied as: effective_register = register - offset

# Offset semantics:
# - Offset is applied per slave, not per device.
# - Effective Modbus address is computed as:
#     effective_register = register - offset
# - Offset depends on how the vendor documents register maps.
#   Some devices mix 0-based and 1-based addressing across register ranges.
# - It is the user's responsibility to set the correct offset for each slave.

# Words vs datatype:
# - words must match the size required by datatype:
#     * 1 word  -> U8, U16, S16
#     * 2 words -> U32, S32, U32LE, F32*
#     * 4 words -> U64BE, S64BE, F64BE
# - Mismatched word counts may result in incorrect values.

# UTF8 datatype:
# - UTF8 registers are read and decoded correctly,
#   but are NOT exported as Prometheus metrics.
# - Prometheus only supports numeric samples.
# - UTF8 registers may be used for debugging or future metadata endpoints.

# Gain:
# - Gain is applied after decoding the raw register value.
# - Final value = decoded_value * gain
# - Use gain to apply scaling factors documented by the vendor.


poll_interval: 60s

devices:
  - name: "ion_7400"
    protocol: "modbus-tcp"
    address: "192.168.1.3"
    port: 502
    timeout: 1s

    slaves:
      - name: "ion_7400_1"
        slave_id: 1
        offset: 1
        modbus_registers:
          - register: 3110
            function_code: 3
            name: "frequency"
            description: "frequency"
            words: 2
            datatype: "F32BE"
            unit: "Hz"
            gain: 1
            flags:

  - name: "ion_7400_2"
    protocol: "modbus-tcp"
    address: "192.168.1.4"
    port: 502
    timeout: 1s

    slaves:
      - name: "ion_7400_2"
        slave_id: 1
        offset: 1
        modbus_registers:
          - register: 3110
            function_code: 3
            name: "frequency"
            description: "frequency"
            words: 2
            datatype: "F32BE"
            unit: "Hz"
            gain: 1
            flags:

  - name: "logger3000"
    protocol: "modbus-tcp"
    address: "192.168.1.205"
    port: 502
    timeout: 1s

    slaves:
      - name: "SG250HX_1"
        slave_id: 1
        offset: 1
        modbus_registers:
          - register: 5008             
            function_code: 4
            name: "internal_temperature"
            description: "Internal temperature"
            words: 1
            datatype: "S16"
            unit: "°C"
            gain: 0.1

      - name: "SG250HX_2"
        slave_id: 2
        offset: 1
        modbus_registers:
          - register: 5008
            function_code: 4
            name: "internal_temperature"
            description: "Internal temperature"
            words: 1
            datatype: "S16"
            unit: "°C"
            gain: 0.1

  - name: "logger3000_2"
    protocol: "modbus-tcp"
    address: "192.168.1.206"
    port: 502
    timeout: 1s

    slaves:
      - name: "SG250HX_8"
        slave_id: 1
        offset: 1
        modbus_registers:
          - register: 5035             
            function_code: 4
            name: "power_factor"
            description: "Power factor"
            words: 1
            datatype: "S16"
            unit: ""
            gain: 0.001
            flags:

      - name: "SG250HX_102"
        slave_id: 3
        offset: 1
        modbus_registers:
          - register: 5001
            function_code: 4
            name: "nominal_active_power"
            description: "Nominal active power"
            words: 1
            datatype: "U16"
            unit: "kW"
            gain: 0.1
            flags:
```

---

## HTTP endpoints

- `/metrics` — Prometheus metrics
- `/health` — process liveness

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
      - "your-config.yml"
      - "--debug"

    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:9105/health"]
      interval: 30s
      timeout: 5s
      retries: 3
```

---

## Troubleshooting

- `/health` OK but no metrics:
  - verify IP / port / unit_id
  - check firewall rules
  - run with --debug

- Timeouts:
  - device offline
  - wrong network route
  - Modbus port blocked

- Unexpected values:
  - verify register address and type
  - check scaling factor
  - check byte order

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
- Monitor modbus_sample_age_seconds to detect stale devices.

---

## Docker images

Images are published in Docker Hub as:

atrabilis/modbus-exporter:<tag>

Tags:

- v0.1.0 → release
- latest → latest stable
- test → development only

---

## Upgrade procedure

1) Update docker-compose image tag.
2) Pull new image:

```bash
docker compose pull
```

3) Restart:

```bash
docker compose up -d
```

---

## Rollback

Revert the image tag to the previous version and restart the container.

---

## License

MIT
