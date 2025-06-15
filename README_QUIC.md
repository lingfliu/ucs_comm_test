# QUIC Pingpong Latency Test

This QUIC implementation provides a modern, secure pingpong test to measure network latency between client and server using the QUIC protocol.

## Features

- **QUIC Server**: Listens on a specified port and echoes back received data over QUIC streams
- **QUIC Client**: Sends pingpong packets at a specified rate and measures round-trip latency
- **Built-in TLS**: QUIC includes TLS 1.3 encryption by default
- **Stream-based**: Uses QUIC streams for reliable, ordered delivery
- **Low Latency**: QUIC's 0-RTT connection establishment reduces handshake overhead
- **Configurable Rate**: Adjustable packet sending frequency (fps)

## Usage

### Start the QUIC Server

```bash
# Default port 10074
go run ./mock/quic_srv/quic_srv.go

# Custom port
go run ./mock/quic_srv/quic_srv.go --host_port 10075
```

### Start the QUIC Client

```bash
# Default settings (127.0.0.1:10074, 10 fps)
go run ./mock/quic_cli/quic_cli.go

# Custom settings
go run ./mock/quic_cli/quic_cli.go --host_addr 192.168.1.100 --host_port 10075 --fps 5
```

## Parameters

### Server Parameters
- `--host_port`: Port to listen on (default: 10074)

### Client Parameters
- `--host_addr`: Server IP address (default: 127.0.0.1)
- `--host_port`: Server port (default: 10074)
- `--fps`: Packets per second to send (default: 10)

## Example Output

**Server Output:**
```
{"time":"2025-06-15T23:15:00.000Z","level":"INFO","msg":"[quic_srv] starting listening at port 10074"}
{"time":"2025-06-15T23:15:01.000Z","level":"INFO","msg":"[quic_accept] new connection from 127.0.0.1:xxxxx"}
{"time":"2025-06-15T23:15:01.000Z","level":"INFO","msg":"[quic_srv] new client connected"}
{"time":"2025-06-15T23:15:01.000Z","level":"INFO","msg":"[quic_srv] received 16 bytes, echoing back"}
```

**Client Output:**
```
connecting to 127.0.0.1:10074
connected, start pingpong at fps = 10
{"time":"2025-06-15T23:15:01.000Z","level":"INFO","msg":"[quiccli] sending pingpong idx = 0"}
{"time":"2025-06-15T23:15:01.000Z","level":"INFO","msg":"[quiccli] recv pingpong idx = 0, latency = 3"}
```

## Advantages of QUIC

1. **Built-in Security**: TLS 1.3 encryption is mandatory
2. **Connection Migration**: Can survive network changes (IP/port changes)
3. **Multiplexing**: Multiple streams without head-of-line blocking
4. **0-RTT Resumption**: Faster reconnection for repeat connections
5. **Congestion Control**: Advanced congestion control algorithms
6. **UDP-based**: Avoids TCP's limitations while providing reliability

## Protocol Comparison

| Feature | TCP | UDP | QUIC |
|---------|-----|-----|------|
| **Port** | 10071 | 10072 | 10074 |
| **Reliability** | Reliable | Unreliable | Reliable |
| **Encryption** | Optional | None | Built-in TLS 1.3 |
| **Connection** | Stream-based | Connectionless | Stream-based |
| **Handshake** | 3-way | None | 1-RTT or 0-RTT |
| **Latency**: | Medium | Lowest | Low-Medium |
| **Use Case** | General purpose | Low latency testing | Modern applications |

## Protocol Details

The pingpong packet format is identical across all protocols:
- Bytes 0-7: Timestamp (uint64, little-endian)  
- Bytes 8-15: Packet index (uint64, little-endian)

## Security Note

The QUIC implementation uses self-signed certificates with `InsecureSkipVerify: true` for testing purposes. In production, you should use proper certificates and certificate validation. 