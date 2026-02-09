# avular-packages

Unified dependency resolver and packager for Avular Robotics projects. Resolves dependencies from ROS `package.xml` files and product specifications, producing deterministic lockfiles and deb packages for deployment.

## Overview

`avular-packages` provides a single runtime channel for all dependencies using `dpkg/deb` as the package format. Python dependencies are packaged as debs, eliminating the need for pip at runtime. The system supports product-level specifications with profile composition, explicit packaging modes, and deterministic resolution.

## Features

- Unified dependency resolution from ROS `package.xml` export tags and product specs
- Deterministic lockfile generation with snapshot identifiers
- Python dependency packaging as deb files
- Product spec composition with profile overrides
- Explicit packaging modes: `individual`, `meta-bundle`, `fat-bundle`
- ProGet Debian feed integration with snapshot-style distributions
- Compatibility wrappers for legacy `get-dependencies` workflows

## Installation

### Releases

Download the latest precompiled binary from the [GitHub Releases](https://github.com/avular-robotics/avular-packages/releases) page for your operating system and architecture.

### Build from Source

1. Clone the repository:

```bash
git clone https://github.com/avular-robotics/avular-packages.git
cd avular-packages
```

2. Launch in a devcontainer (recommended):

```bash
just dev # or use your IDE of choice
```

2.5 Install dependencies Locally (optional):

```bash
just deps
```

3. Build the binary:

```bash
just build
```

## Quick Start

1. Create a product specification:

```yaml
api_version: "v1"
kind: "product"
metadata:
  name: "my-product"
  version: "2026.01.27"
compose:
  - name: "base-profile"
    version: "2026.01"
    source: "local"
    path: "profile.yaml"
inputs:
  package_xml:
    enabled: true
    tags: ["debian_depend", "pip_depend"]
```

2. Validate the specification:

```bash
avular-packages validate --product product.yaml --profile profile.yaml
```

3. Resolve dependencies:

```bash
avular-packages resolve \
  --product product.yaml \
  --profile profile.yaml \
  --workspace ./src \
  --repo-index repo-index.yaml \
  --output ./out
```

4. Build deb packages:

```bash
avular-packages build \
  --product product.yaml \
  --profile profile.yaml \
  --workspace ./src \
  --repo-index repo-index.yaml \
  --output ./out \
  --debs-dir ./debs
```

## Commands

- `validate` - Validate product and profile specifications
- `resolve` - Resolve dependencies and produce lockfiles
- `build` - Build deb artifacts from resolved dependencies
- `publish` - Publish artifacts to ProGet Debian feed
- `inspect` - Inspect resolved dependency graphs
- `repo-index` - Build repository index files
- `lock` - Generate lockfiles without full resolution

## Architecture

The system uses hexagonal architecture with explicit ports and adapters. Core services include:

- **SpecCompiler**: Validates and normalizes product specs
- **ProductComposer**: Composes profile layers into product view
- **ResolverCore**: Resolves dependencies and conflicts

## Requirements

- Go 1.25 or later
- Ubuntu-based system for runtime
- Access to ProGet Debian feed (for publishing)

## Documentation

See `docs/` for detailed specifications and design documents.
