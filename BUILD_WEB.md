# Web Application Build Guide

This document explains how to build and embed the React/TypeScript web application into the Go binary.

## Prerequisites

Make sure you have Node.js and npm installed:
```bash
node --version
npm --version
```

## Build Process

### 1. Install Web Dependencies

First, install the React application dependencies:
```bash
make install-web
```

Or manually:
```bash
cd web && npm install
```

### 2. Build React Application

For production build:
```bash
make build-web
```

For development build:
```bash
make build-web-dev
```

### 3. Build Go Binary with Embedded Assets

Build the Go server with embedded React application:
```bash
make build-embedded
```

This will:
1. Build the React application using Vite
2. Copy the built assets to `pkg/embedded/web/dist/`
3. Build the Go binary with embedded assets using `go:embed`

### 4. Run the Server

Start the server with embedded React application:
```bash
make server
```

Or run the binary directly:
```bash
./bin/nala-coder-server
```

## Development Workflow

### Full Clean Build
```bash
make clean
make server
```

### Quick Rebuild (after web changes)
```bash
make build-web
make build-embedded
./bin/nala-coder-server
```

## File Structure

```
web/                    # React/TypeScript source code
├── src/               # React components and logic
├── public/            # Static assets
├── dist/              # Build output (generated)
├── package.json       # Node.js dependencies
└── vite.config.ts     # Vite build configuration

pkg/embedded/
├── files.go           # Go embed directives
└── web/
    └── dist/          # Copied React build output (generated)
```

## How It Works

1. **Vite Build**: The React application is built using Vite, which creates optimized static files in `web/dist/`

2. **Asset Copying**: The Makefile copies the built assets to `pkg/embedded/web/dist/`

3. **Go Embed**: The Go `embed` directive in `pkg/embedded/files.go` embeds all files from `web/dist/` into the binary

4. **HTTP Routing**: The Gin router serves the embedded static files and handles React Router's client-side routing

## Troubleshooting

### Build Fails
- Make sure Node.js and npm are installed
- Run `make install-web` to install dependencies
- Check that `web/package.json` exists

### Assets Not Loading
- Verify that `web/dist/` contains the built files
- Check that `pkg/embedded/web/dist/` has the copied assets
- Ensure the Go embed directive points to the correct path

### React Router Issues
- The server is configured to serve `index.html` for all non-API routes
- This enables client-side routing for the React SPA

## Configuration

### Vite Configuration (`web/vite.config.ts`)
- Output directory: `dist/`
- Base path: `/`
- Assets directory: `assets/`

### Go Embed Configuration (`pkg/embedded/files.go`)
- Embed path: `web/dist`
- Serves static assets from `/assets/`
- Serves `index.html` for SPA routing