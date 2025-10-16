# SeeMUD - Visual MUD Client

**Transform classic text-based MUDs into immersive visual experiences with AI-powered image generation.**

SeeMUD is an innovative MUD (Multi-User Dungeon) client that enhances traditional text-based gaming by automatically generating visual representations of game environments while preserving the classic gameplay experience.

![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20Windows%20%7C%20macOS-lightgrey)
![Go Version](https://img.shields.io/badge/Go-1.23-blue)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

- **AI-Powered Text Parsing** - Intelligent classification of room descriptions, items, and NPCs
- **Dynamic Image Generation** - Real-time visualisation of MUD environments using Stable Diffusion
- **Smart Caching** - Intelligent image caching with persistent storage across sessions
- **Auto-Mapping** - Automatic spatial mapping with 2-level neighbourhood awareness
- **Spatial Context** - Image generation leverages neighbouring room context for consistency
- **Three-Column Layout** - Independent resizable panels for output, map, and image display
- **Entity Detection** - Automatic identification and display of items and mobs in rooms

## Screenshots

*Coming soon - showing the three-column interface with MUD output, mini-map, and AI-generated room imagery*

## Quick Start

### Prerequisites

- Go 1.23 or higher
- Node.js 16+ and npm (for frontend development)
- [Wails v2](https://wails.io/docs/gettingstarted/installation) installed
- Stable Diffusion API endpoint (default: `http://127.0.0.1:7860`)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/seeMUD.git
cd seeMUD

# Install dependencies
go mod download

# Build the application
wails build

# Or run in development mode
wails dev
```

### Configuration

Set the Stable Diffusion endpoint via environment variable:

```bash
export SEEMUD_SD_ENDPOINT="http://localhost:7860"
```

Or the endpoint will default to `http://127.0.0.1:7860`

### Running

After building, run the binary:

```bash
./build/bin/seemud-gui  # Linux/macOS
# or
./build/bin/seemud-gui.exe  # Windows
```

Or use the convenience script:

```bash
./launch.sh  # Development mode
./launch-binary.sh  # Production binary
```

## Usage

1. **Connect to MUD** - Enter host and port, click Connect
2. **Explore** - Move through rooms as normal (north, south, east, west, etc.)
3. **View Map** - Auto-generated mini-map shows current room and surroundings
4. **Generate Images** - Click "Generate Image" to visualise the current room
5. **Regenerate** - Don't like the image? Click "Regenerate" for a new version
6. **Custom Prompts** - Add custom style directions when regenerating images

## Architecture

SeeMUD uses a clean separation of concerns:

- **Telnet Client** - Handles MUD server connections
- **Parser** - Classifies MUD output (room titles, descriptions, exits, entities)
- **Mapper** - Builds spatial graph of rooms with intelligent duplicate handling
- **Renderer** - Generates images using Stable Diffusion with contextual prompts
- **Frontend** - React-based UI built with Wails framework

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

## Project Structure

```
seeMUD/
├── internal/
│   ├── telnet/          # MUD connection handling
│   ├── parser/          # Text parsing and classification
│   ├── mapper/          # Spatial mapping and graph building
│   └── renderer/        # Image generation (Stable Diffusion)
├── frontend/            # React UI
│   └── src/
│       ├── App.jsx      # Main application
│       ├── Terminal.jsx # MUD output display
│       └── Map.jsx      # Mini-map visualisation
├── docs/                # Comprehensive documentation
├── cache/               # Generated image cache
└── maps/                # Saved map data
```

## Documentation

Comprehensive documentation is available in the `docs/` directory:

- [Product Requirements](docs/PRD.md) - Vision and feature specifications
- [Architecture](docs/ARCHITECTURE.md) - System design and component overview
- [Protocol](docs/PROTOCOL.md) - SeeMUD protocol specification
- [Adapter Design](docs/ADAPTER_DESIGN.md) - MUD adapter implementation guide
- [AI Integration](docs/AI_INTEGRATION.md) - AI classification strategy
- [Roadmap](docs/ROADMAP.md) - Development phases and milestones

## Current Status

**Active Development** - Core functionality implemented:

- ✅ MUD connection and telnet handling
- ✅ Text parsing for WolfMUD format
- ✅ Spatial mapping with duplicate room handling
- ✅ Image generation with neighbour context
- ✅ Image caching and persistence
- ✅ Three-column resizable UI
- ✅ Item and mob detection
- 🚧 AI-powered adapter system (planned)
- 🚧 Multi-MUD support (planned)
- 🚧 Advanced compositing (planned)

## Development

### Building from Source

```bash
# Development mode with hot reload
wails dev

# Production build
wails build

# Clean build
wails build -clean
```

### Frontend Development

```bash
cd frontend
npm install
npm run dev
```

### Running Tests

```bash
go test ./...
```

## Configuration Files

- `wails.json` - Wails project configuration
- `go.mod` - Go module dependencies
- `frontend/package.json` - Frontend dependencies

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues for bugs and feature requests.

## License

This project is licensed under the MIT Licence - see the LICENSE file for details.

## Acknowledgements

- Built with [Wails](https://wails.io/) - Go + Web UI framework
- Image generation via [Stable Diffusion](https://github.com/AUTOMATIC1111/stable-diffusion-webui)
- Inspired by classic MUDs and the golden era of text-based gaming

## Support

For issues, questions, or suggestions:
- Open an issue on GitHub
- Check existing documentation in `docs/`
- Review the architecture guide for technical details

---

**Note:** This is a client-side application that connects to existing MUD servers. No server modifications are required.
