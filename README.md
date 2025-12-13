# System Updates Monitor

A distributed system monitoring tool that tracks pending package updates across your home network. The system consists of client that run on each machine and a server that aggregates and displays update information via a web interface.

## Overview

This project monitors system updates across multiple machines in your network:
- **Client**: Runs on each machine (macOS, Linux) and periodically checks for pending updates
- **Server**: Aggregates update information from all clients and provides a web dashboard
- **Communication**: Uses NATS for messaging between clients and server

## Supported Systems

### Package Managers
- **Linux**: apt (Debian/Ubuntu), dnf (Fedora/RHEL), yum (older RHEL/CentOS)
- **macOS**: Homebrew (brew)

### Platforms
- macOS (darwin)
- Linux (ARM, ARM64, x86_64)

## Architecture

- **Clients** connect to a NATS server and publish system information including pending updates every minute
- **Server** can run an embedded NATS server (default) or connect to an external NATS instance
- **Storage** uses BoltDB to persist system state
- **Web Interface** provides a real-time dashboard to view all systems and their update status

## Building

### Build All Components
```bash
make build
```

### Build Individual Components
```bash
make client    # Build client only
make server    # Build server only
```

### Cross-Compilation
```bash
make linux     # Build Linux binaries for all architectures (ARM, ARM64, AMD64)
make linux-all # Build both client and server for all Linux architectures
```

### Other Targets
```bash
make help      # Show all available make targets
make clean     # Clean build artifacts
make test      # Run tests
make fmt       # Format code
```

## Configuration

### Server Configuration

The server can be configured via environment variables:

- `NATS_URL`: NATS server URL (default: `embedded` - runs embedded NATS server)
- `NATS_PORT`: Port for embedded NATS server (default: `4222`)
- `DB_PATH`: Path to BoltDB database file (default: `systems.db`)
- `HTTP_PORT`: Web server port (default: `8080`)

Example:
```bash
export HTTP_PORT=3000
export DB_PATH=/var/lib/updateHUD/systems.db
./server
```

### Client Configuration

The client can be configured via environment variables:

- `NATS_URL`: NATS server URL (e.g., `nats://192.168.1.157:4222`)
- `NATS_SERVER_IP`: NATS server IP address (default: `192.168.1.157`)
- `DEBUG`: Enable debug logging (set to `1` or `true`)

Example:
```bash
export NATS_URL=nats://192.168.1.157:4222
./client
```

### Running as a Service

The client can be run as a systemd service. See `client/contrib/systemd.unit` for an example systemd unit file.

## Usage

1. **Start the server**:
   ```bash
   cd server
   make run
   ```
   The web interface will be available at `http://localhost:8080` (or your configured HTTP_PORT).

2. **Run clients on each machine**:
   ```bash
   cd client
   ./client
   ```
   The client will automatically connect to the NATS server and start reporting system information.

3. **View the dashboard**: Open your browser to the server's HTTP port to see all systems and their update status.

## Web Interface

The web dashboard provides:
- Overview of all monitored systems
- System details (hostname, OS, architecture, IP address)
- Pending update lists with package names and versions
- Sortable columns
- Expandable rows to view detailed update information
- Last seen timestamps

## Alternatives

Instead of using this tool, you could run a cron job or systemd timer to auto-update. However, this approach has drawbacks:
- Sometimes reboots are required after updates
- Services (like Docker) may crash during updates
- You lose visibility and control over when updates are applied

This tool gives you visibility into pending updates across all your systems, allowing you to plan updates appropriately.
