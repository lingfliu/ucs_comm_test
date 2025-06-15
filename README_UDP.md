# UDP Pingpong Latency Test

This UDP implementation provides a simple pingpong test to measure network latency between client and server.

## Features

- **UDP Server**: Listens on a specified port and echoes back received data
- **UDP Client**: Sends pingpong packets at a specified rate and measures round-trip latency
- **Latency Measurement**: Calculates network latency in milliseconds
- **Configurable Rate**: Adjustable packet sending frequency (fps)

## Usage

### Start the UDP Server

```bash
# Default port 10072
go run ./mock/udp_srv/udp_srv.go

# Custom port
go run ./mock/udp_srv/udp_srv.go --host_port 10073
```

### Start the UDP Client

```bash
# Default settings (127.0.0.1:10072, 10 fps)
go run ./mock/udp_cli/udp_cli.go

# Custom settings
go run ./mock/udp_cli/udp_cli.go --host_addr 192.168.1.100 --host_port 10073 --fps 5
```

## Parameters

### Server Parameters
- `--host_port`: Port to listen on (default: 10072)

### Client Parameters
- `--host_addr`: Server IP address (default: 127.0.0.1)
- `--host_port`: Server port (default: 10072)
- `--fps`: Packets per second to send (default: 10)

## Example Output

**Server Output:**
```
{"time":"2025-06-15T22:55:00.000Z","level":"INFO","msg":"[udp_srv] starting listening at port 10072"}
{"time":"2025-06-15T22:55:01.000Z","level":"INFO","msg":"[udp_srv] new client connected"}
{"time":"2025-06-15T22:55:01.000Z","level":"INFO","msg":"[udp_srv] received 16 bytes, echoing back"}
```

**Client Output:**
```
connecting to 127.0.0.1:10072
connected, start pingpong at fps = 10
{"time":"2025-06-15T22:55:01.000Z","level":"INFO","msg":"[udpcli] sending pingpong idx = 0"}
{"time":"2025-06-15T22:55:01.000Z","level":"INFO","msg":"[udpcli] recv pingpong idx = 0, latency = 2"}
```

## Differences from TCP Version

1. **Port**: UDP uses port 10072 by default (TCP uses 10071)
2. **Connection Model**: UDP is connectionless, so no persistent connection state
3. **Reliability**: UDP doesn't guarantee packet delivery (unlike TCP)
4. **Performance**: Generally lower latency due to reduced protocol overhead

## Protocol

The pingpong packet format is:
- Bytes 0-7: Timestamp (uint64, little-endian)
- Bytes 8-15: Packet index (uint64, little-endian)

The client sends packets with current timestamp and incremental index, and the server echoes them back unchanged. The client calculates latency as the difference between current time and the original timestamp. 