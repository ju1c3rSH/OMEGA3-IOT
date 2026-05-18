# OMEGA3-IOT Debugger

A web-based debugging interface for the OMEGA3-IOT platform.

## Structure

```
debugger/
├── index.html          # Main HTML file
├── css/
│   ├── styles.css      # Application styles
│   └── fonts.css       # Font declarations (local + CDN fallback)
├── js/
│   └── app.js          # Application JavaScript
├── assets/
│   └── fonts/          # Cached font files (local)
├── download-fonts.ps1  # Script to download fonts for offline use
└── README.md           # This file
```

## Usage

### Quick Start

1. Open `index.html` in a web browser
2. The debugger will load fonts from CDN if local cache is not available

### Offline Use (Recommended)

To use the debugger without internet connectivity:

1. Open PowerShell in the `debugger` folder
2. Run the font download script:
   ```powershell
   .\download-fonts.ps1
   ```
3. Fonts will be cached in `assets/fonts/` for offline use

### Features

- **User Authentication**: Register and login to get JWT tokens
- **Device Management**: View and manage IoT devices
- **Device Registration**: Anonymous registration and user binding
- **Device Groups**: Create and manage device groups
- **Command Sending**: Send actions to devices via MQTT
- **History Queries**: Query historical device data
- **System Logs**: View device and user logs
- **Device Sharing**: Share devices with other users

## Configuration

The debugger connects to the OMEGA3-IOT API. You can configure the API URL in the header input field (default: `http://localhost:27015`).

## Browser Compatibility

- Chrome 80+
- Firefox 75+
- Safari 13+
- Edge 80+

## Development

The debugger is built with vanilla HTML, CSS, and JavaScript. No build tools or frameworks are required.

To modify the debugger:

1. Edit `css/styles.css` for styling changes
2. Edit `js/app.js` for functionality changes
3. Edit `index.html` for structure changes
