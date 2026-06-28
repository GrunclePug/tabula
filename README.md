# tabula

tabula - A minimalist, self-hosted developer dashboard and notes navigator.

## Overview

`tabula` is a lightweight, hypermedia-driven dashboard designed for developers to aggregate local services, repository documentation, and markdown notes. Built with Go and HTMX, it operates as a single-binary compilation with embedded assets, making it exceptionally easy to host locally, via Docker Compose, or inside lightweight Kubernetes clusters (k3s).

## Features

* **Zero-CGO Pure Go**: Statically compiled using Go's standard library `http.ServeMux` and a CGO-free SQLite driver (`modernc.org/sqlite`) for absolute portability.
* **Hypermedia-Driven UI**: Utilizes HTMX to perform dynamic component updates, inline configuration, group collapsing, and renaming without any Javascript framework overhead.
* **Self-Contained Executable**: Embeds all HTML templates, Tailwind CSS sheets, and static scripts directly into the binary via `go:embed`, enabling a true single-file deployment.
* **Interactive Drag & Drop**: Implements native HTML5 drag-and-drop to reorder service cards inside groups, transfer cards between groups, and save positions silently in the background.
* **Markdown Navigator**: Links directly to a local directory (e.g. an Obsidian vault) to search, autocomplete, parse (using Goldmark), and read documents inline inside overlay modals.
* **Ingestion Pipeline**: Paste screenshots directly (`Ctrl+V` or clipboard button) or drag-and-drop file uploads to dynamically save icon images with collision-free filenames.
* **Container-Ready**: Minimalist multi-stage `Dockerfile` compilation running on a Distroless runtime image with native environment-based bind mounts.

## Usage

You can run the application directly from the compiled binary or deploy it via container stacks.

### Local Execution

```bash
# Copy and configure environment variables
cp .env.example .env

# Compile tailwind styles and build the Go executable
make build

# Run the local dashboard server (defaults to port 8080)
make run
```

### Docker Compose

```bash
# Start the container stack in the background (binds to host port 80)
docker compose up -d

# View logs
docker compose logs -f

# Shut down the stack
docker compose down
```

### Requirements

* Go 1.22+
* Make
* Docker / Docker Compose (optional for container hosting)

### Configuration

Customize options inside your local `.env` file:
```env
# Absolute path to your Obsidian/markdown notes vault root
TABULA_DOCS_ROOT="/home/user/Sync/Vault"

# Isolated directory to store the database and dynamic user uploads
TABULA_DATA_DIR=data

# External port mapping (defaults to 8080 if running binary, 80 for docker compose)
PORT=80
```

### Maintenance

Clean build artifacts and downloaded compiler binaries:
```bash
make clean
```

## Contributing

Contributions are welcome! If you'd like to contribute to this project, please fork the repository and submit a pull request.

## Author

Chad Humphries |
[Website](https://grunclepug.com/) |
[GitHub Profile](https://github.com/GrunclePug)

## Other Projects

Check out some of my other projects on GitHub: [Here](https://github.com/GrunclePug?tab=repositories)
