# Elephant 🐘

**Elephant** - _cuz it's phat_ - is a powerful data provider service and backend for building custom application launchers and desktop utilities. It provides various data sources and actions through a plugin-based architecture, communicating via Unix sockets and Protocol Buffers.

[![Discord](https://img.shields.io/discord/1402235361463242964?logo=discord)](https://discord.gg/mGQWBQHASt)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

## Overview

Elephant acts as a unified backend service that aggregates data from various sources (desktop applications, files, clipboard history, etc.) and provides a consistent interface for frontend applications like custom launchers, productivity tools, or desktop widgets.

## Features

### Current Providers

- **🚀 Desktop Applications**
  - Auto-detection of `uwsm` or `app2unit`
  - Application launch history
  - Desktop entry parsing

- **📁 Files**
  - File preview (text/image)
  - File operations: open, copy path, copy content
  - Directory navigation

- **📋 Clipboard**
  - Text and image clipboard history
  - Clipboard content management

- **⚡ Runner**
  - Command execution from explicit lists
  - `$PATH` scanning for executables

- **🔣 Symbols/Emojis**
  - Multi-locale emoji and symbol support
  - Unicode character database

- **🧮 Calculator/Unit Conversion**
  - Mathematical calculations with history
  - Unit conversion using `qalc`

- **📋 Custom Menus**
  - User-defined menu creation
  - Custom action definitions

- **📊 Provider List**
  - Dynamic listing of all loaded providers and menus

- **🌐 Websearch**
  - ... define custom search engines

- **🌐 Archlinux Packages**
  - list all packages: official and AUR
  - list all explicitly installed packages `i:`
  - remove / install packages (pacman, yay/paru)

## Installation

### Installing on Arch

```
yay -S elephant

# Providers, f.e.

yay -S elephant-desktopapplications
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/abenz1267/elephant
cd elephant

# Build and install the main binary
cd cmd
go install elephant.go

# Create configuration directories
mkdir -p ~/.config/elephant/providers

# Build and install a provider (example: desktop applications)
cd ../internal/providers/desktopapplications
go build -buildmode=plugin
cp desktopapplications.so ~/.config/elephant/providers/
```

## Usage

### Starting the Service

```bash
# Start elephant with default configuration
elephant

# Start with debug logging
elephant --debug

# Use custom configuration directory
elephant --config /path/to/config
```

### Command Line Interface

Elephant includes a built-in client for testing and basic operations:

#### Querying Data

```bash
# Query provider (providers;query;limit;exactsearch)
elephant query "files;documents;10;false"
```

#### Activating Items

```bash
# activate item (qid;provider;identifier;action;query)
elephant activate "1;files;<identifier>;open;"
```

#### Other Commands

```bash
# List all installed providers
elephant listproviders

# Open a custom menu, requires a subscribed frontend.
elephant menu "screenshots"

# Show version
elephant version

# Generate configuration documentation
elephant generatedoc
```

### Configuration

Elephant uses a configuration directory structure:

```
~/.config/elephant/
├── elephant.toml        # Main configuration
├── .env                 # Environment variables
└── providers/           # Provider plugins
    ├── files.so
    ├── desktopapplications.so
    └── ...
```

## API & Integration

### Communication Protocol

Elephant uses Unix domain sockets for IPC and Protocol Buffers for message serialization. The main message types are:

- **Query Messages**: Request data from providers
- **Activation Messages**: Execute actions
- **Menu Messages**: Request custom menu data
- **Subscribe Messages**: Listen for real-time updates

### Building Client Applications

To integrate with Elephant, your application needs to:

1. Connect to the Unix socket (typically at `/tmp/elephant.sock`)
2. Send Protocol Buffer messages
3. Handle responses and updates

See the `pkg/pb/` directory for Protocol Buffer definitions.

## Development

### Project Structure

```
elephant/
├── cmd/                 # Main application entry point
├── internal/
│   ├── comm/           # Communication layer (Unix sockets, protobuf)
│   ├── common/         # Shared utilities and configuration
│   ├── providers/      # Data provider plugins
│   └── util/          # Helper utilities
├── pkg/pb/            # Protocol Buffer definitions
└── flake.nix          # Nix development environment
```

### Creating Custom Providers

Providers are Go plugins that implement the provider interface. See existing providers in `internal/providers/` for examples.

### Building from Source

```bash
# Clone repository
git clone https://github.com/abenz1267/elephant
cd elephant

# Install dependencies
go mod download

# Build main binary
go build -o elephant cmd/elephant.go

# Run tests
go test ./...
```

### Development Environment

A Nix flake is provided for reproducible development:

```bash
nix develop
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

Please ensure your code follows Go best practices and includes appropriate documentation.

## Current Status

⚠️ **Work in Progress**: This project is in active development and the API may change. Use with caution in production environments.

## License

This project is licensed under the GNU General Public License v3.0. See [LICENSE](LICENSE) for details.

## Support

- 💬 [Discord Community](https://discord.gg/mGQWBQHASt)
- 🐛 [Issue Tracker](https://github.com/abenz1267/elephant/issues)
