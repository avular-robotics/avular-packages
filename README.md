# avular-packages

Unified dependency resolver and packager for Avular Robotics projects. Resolves dependencies from ROS `package.xml` files and product specifications, producing deterministic lockfiles and deb packages for deployment.

## Overview

`avular-packages` provides a single runtime channel for all dependencies using `dpkg/deb` as the package format. Python dependencies are packaged as debs, eliminating the need for pip at runtime. The system supports product-level specifications with profile composition, schema-driven ROS tag resolution, explicit packaging modes, and deterministic resolution.

## Features

- **Dependency resolution** from ROS `package.xml` export tags, standard ROS tags (`<depend>`, `<exec_depend>`, `<build_depend>`), and manual product/profile specs
- **Schema-driven resolution** -- map abstract ROS dependency keys to concrete apt/pip packages using layered `schema.yaml` files
- **Deterministic lockfiles** with snapshot identifiers and SBOM generation (SPDX-2.3)
- **Python-to-deb packaging** -- Python dependencies built as `.deb` files, no pip at runtime
- **Product/profile composition** -- layer profile specs onto a product spec with inline schema overrides
- **Packaging modes** -- `individual`, `meta-bundle`, and `fat-bundle`
- **Repository backends** -- local file, Aptly, and ProGet Debian feed integration
- **Repository index builder** -- fetch and index APT and PyPI feeds with caching and retry
- **Snapshot lifecycle** -- publish, promote, prune with configurable retention policies
- **Compatibility wrappers** for legacy `get-dependencies` and `rosdep` workflows

## Installation

### Releases

Download the latest precompiled binary from the [GitHub Releases](https://github.com/avular-robotics/avular-packages/releases) page for your operating system and architecture.

### Build from Source

Clone the repository:

```bash
git clone https://github.com/avular-robotics/avular-packages.git
cd avular-packages
```

Launch in a devcontainer (recommended):

```bash
just dev
```

Or install dependencies locally:

```bash
just deps
```

Build, test, and lint:

```bash
just check    # fmt + lint + test
```

The binary lands in `./out/` by default.

## Quick Start

**1. Scaffold a new product spec:**

```bash
avular-packages init --name my-product --dir .
```

**2. Validate specs:**

```bash
avular-packages validate --product product.yaml --profile profile.yaml
```

**3. Resolve dependencies:**

```bash
avular-packages resolve \
  --product product.yaml \
  --workspace ./src \
  --repo-index repo-index.yaml \
  --target-ubuntu 24.04 \
  --output ./out
```

**4. Build deb packages:**

```bash
avular-packages build \
  --product product.yaml \
  --workspace ./src \
  --repo-index repo-index.yaml \
  --target-ubuntu 24.04 \
  --output ./out \
  --debs-dir ./debs
```

**5. Publish to a repository:**

```bash
avular-packages publish \
  --output ./out \
  --debs-dir ./debs \
  --repo-backend proget \
  --proget-endpoint https://proget.example.com \
  --proget-feed apt-releases \
  --proget-api-key "$PROGET_API_KEY"
```

## Commands

| Command | Description |
|---|---|
| `init` | Scaffold a new product spec with sensible defaults |
| `validate` | Validate product and profile specifications |
| `resolve` | Resolve dependencies and produce lockfiles |
| `lock` | Alias for `resolve` |
| `build` | Build deb artifacts from resolved dependencies |
| `publish` | Publish artifacts and create a snapshot |
| `inspect` | Inspect resolved outputs and bundle membership |
| `repo-index` | Generate a repository index from APT and PyPI feeds |
| `prune` | Prune snapshot distributions based on retention policy |

Run `avular-packages <command> --help` for flag details.

## Schema Resolution

Standard ROS `package.xml` tags like `<depend>`, `<exec_depend>`, and `<build_depend>` declare abstract dependency keys (e.g. `opencv`). Schema files map these keys to concrete, typed packages:

```yaml
schema_version: "1"
mappings:
  opencv:
    type: apt
    package: libopencv-dev
    version: ">=4.5"
  numpy:
    type: pip
    package: numpy
```

Schemas are layered -- later files override earlier ones per key. Pass them on the CLI:

```bash
avular-packages resolve --schema base.yaml --schema overrides.yaml ...
```

Product and profile specs can also embed schemas inline under `inputs.package_xml.schema`.

## Architecture

The project uses hexagonal architecture (ports and adapters). The core domain is isolated behind interfaces; adapters handle I/O, CLI, and external services.

```
cmd/avular-packages/    Entry point
internal/
  cli/                  Cobra command definitions
  app/                  Application services (orchestration)
  core/                 Domain logic (resolver, builder, composer, SAT solver)
  policies/             Conflict and packaging policy engines
  ports/                Interface contracts
  adapters/             Implementations (file I/O, HTTP, ProGet, Aptly, SBOM)
  shared/               Common utilities
  types/                Domain types
tests/
  e2e/                  End-to-end tests
  integration/          Integration + golden file tests
  testutil/             Shared test helpers
fixtures/               Sample specs, repo indices, workspaces
docs/                   Design documents and specifications
```

### Key Domain Concepts

- **ProductComposer** -- merges product and profile specs, including inline schemas
- **DependencyBuilder** -- collects dependencies from manual entries, `package.xml` typed tags, and schema-resolved ROS tags
- **ResolverCore** -- resolves dependencies against a repo index, applies conflict policies, optionally runs an APT SAT solver
- **PackagePolicy** -- assigns packaging groups and modes to resolved dependencies

## Development

### Prerequisites

- Go 1.25+
- Ubuntu-based system (for deb packaging at runtime)
- `just` task runner (recommended)

### Common Tasks

```bash
just test             # Run tests with race detector
just cover            # Tests + coverage report
just lint             # go vet + golangci-lint
just fmt              # Format source
just check            # All of the above
just clean            # Remove output artifacts
```

### Project Layout Conventions

- **Guard clauses and early returns** -- avoid deep nesting
- **Table-driven tests** with `testify` for assertions
- **Conventional Commits** for git history
- **`errbuilder-go`** for structured, coded errors
- **`zerolog`** for structured logging

## Documentation

The `docs/` directory contains detailed specifications:

- `schema-spec.md` -- schema file format and resolution rules
- `written-spec.md` -- system specification and dependency model
- `resolver-contract.md` -- deterministic output guarantees and conflict handling
- `engineering-design-document.md` -- production design decisions
- `testing.md` -- test strategy, offline fixtures, testcontainers setup
- `justfile-contract.md` -- stable task runner interface
- `devcontainer-spec.md` -- development environment setup

## License

See [LICENSE](LICENSE) for details.
