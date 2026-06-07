# OMEGA3-IOT Debugger

A web-based debugging interface for the OMEGA3-IOT platform, built as a single-file Vue 3 SPA.

## Structure

```
debugger/
├── index.html              # Main debugger (Vue 3 + MQTT.js, single-file SPA)
├── css/                    # Legacy styles (unused by index.html)
├── js/                     # Legacy scripts (unused by index.html)
├── assets/fonts/           # Cached font files for offline use
├── download-fonts.ps1      # Script to download fonts for offline use
└── README.md               # This file
```

## Usage

### Quick Start

Open `index.html` in a web browser. The debugger loads Vue 3 and MQTT.js from CDN (unpkg).

### Serving from Backend

The debugger is also served by the backend at `/debugger` when the application is running.

## Features

### API Tab
- **DH Challenge-Response Auth**: Register/login using the DH protocol (password never transmitted)
- **Device List**: View owned devices with online/offline status
- **Device Registration**: Anonymous registration → reg_code → user binding (two-step flow)
- **Send Commands**: Send actions to devices via HTTP API
- **History Query**: Query IoTDB time-series data with time range and property filters
- **User Groups**: Create, manage, invite, dissolve groups; share devices within groups
- **Device Folders**: Create folders, add/remove devices, organize by category
- **Device Sharing**: Share devices with other users (read/write/read_write permissions)
- **System Logs**: Query device and user operation logs

### Profile Tab
- View/edit user info (nickname, description)
- Avatar upload and reset

### MQTT Simulator Tab
- Simulate IoT devices over MQTT WebSocket
- **4 device types**: BaseTracker, SmartSensor, ToggleMachine, NewsReporter
- Configurable property values with auto-reporting and jitter
- Event sending with severity-based payloads
- Action reception display

### MQTT Debug Tab
- Raw MQTT client for subscribing/publishing
- Message monitoring with topic filtering

### WS Push Tab
- WebSocket client for real-time server push
- ACK-based reliability testing
- Action sending via WebSocket

### Console Panel
- Collapsible bottom panel logging all HTTP, MQTT, and WebSocket traffic
- Filterable by type (HTTP/MQTT/WS)

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| Base URL | `http://lorelei.lat:27015` | API server URL |
| MQTT Broker | `ws://lorelei.lat:9001` | MQTT WebSocket endpoint |
| WS URL | `ws://lorelei.lat:27015/api/v1/ws` | WebSocket push endpoint |

All settings are persisted in localStorage.

## Browser Compatibility

- Chrome 80+
- Firefox 75+
- Safari 13+
- Edge 80+

Requires BigInt support (for DH authentication).

## Development

The debugger is a self-contained single-file application. To modify:

1. Edit `index.html` directly — all HTML, CSS, and JavaScript are inline
2. No build tools, bundlers, or dependencies required
3. Changes take effect on browser refresh
