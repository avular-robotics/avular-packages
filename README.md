# avular-packages

Unified dependency resolver and packager for Avular Robotics projects. Resolves dependencies from ROS `package.xml` files and product specifications, producing deterministic lockfiles and deb packages for deployment.

## Overview

`avular-packages` provides a single runtime channel for all dependencies using `dpkg/deb` as the package format. Python dependencies are packaged as debs, eliminating the need for pip at runtime. The system supports product-level specifications with profile composition, schema-driven ROS tag resolution, explicit packaging modes, and deterministic resolution.

## Features

- **Zero-flag workflows** -- auto-discovers `product.yaml` in the current directory, loads `schemas/` directories automatically, and reads defaults from the product spec so most commands need no flags at all
- **Spec defaults** -- embed `target_ubuntu`, `workspace`, `repo_index`, `output`, `pip_index_url`, and more directly in the product spec; CLI flags and environment variables override when needed
- **Auto-discovery** -- the CLI finds product specs (`product.yaml`, `product.yml`, `avular-product.yaml`) and schema files (`schemas/*.yaml` next to product or profile files) without explicit paths
- **Configuration file** -- project-level `avular-packages.yaml` or `~/.config/avular-packages/avular-packages.yaml` with `AVULAR_PACKAGES_*` environment variable overrides
- **Inline schemas and profiles** -- single-file product specs that embed both packaging policy and schema mappings, no separate files required
- **Schema-driven resolution** -- map abstract ROS dependency keys to concrete apt/pip packages using layered schema files with clear precedence (inline < auto-discovered < spec `schema_files` < CLI `--schema`)
- **Dependency resolution** from ROS `package.xml` export tags, standard ROS tags (`<depend>`, `<exec_depend>`, `<build_depend>`), and manual product/profile specs
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

### 1. Scaffold a product spec

```bash
avular-packages init --name my-product
```

This creates a `product.yaml` in the current directory with inline schema, inline profile, and defaults pre-configured. Edit it to match your project.

### 2. Validate, resolve, and build -- zero flags

Because the product spec carries defaults for `target_ubuntu`, `workspace`, `repo_index`, and `output`, and because the CLI auto-discovers `product.yaml` in the current directory, most commands need no flags:

```bash
# Auto-discovers ./product.yaml, validates inline schemas, profiles, and composition
avular-packages validate

# Auto-discovers ./product.yaml, reads defaults.repo_index, defaults.target_ubuntu, etc.
avular-packages resolve

# Same auto-discovery, delegates to resolve internally, then builds debs
avular-packages build
```

The CLI emits hints to stderr when you pass flags that are already covered by spec defaults, so you can gradually remove flags from your scripts.

### 3. Override when needed

CLI flags always take precedence over spec defaults. Use them for one-off overrides:

```bash
# Override target release for a specific build
avular-packages resolve --target-ubuntu 22.04

# Point at a different repo index
avular-packages build --repo-index /tmp/custom-repo-index.yaml

# Add an extra schema layer on top of auto-discovered ones
avular-packages resolve --schema overrides.yaml
```

### 4. Publish

```bash
avular-packages publish \
  --repo-backend proget \
  --proget-endpoint https://proget.example.com \
  --proget-feed apt-releases \
  --proget-api-key "$PROGET_API_KEY"
```

### How auto-discovery works

The CLI searches for a product spec in this order (first match wins):

1. `product.yaml`
2. `product.yml`
3. `avular-product.yaml`
4. `avular-product.yml`

Schema files are discovered from `schemas/` directories next to the product spec **and** next to each profile file:

```
my-project/
  product.yaml              # auto-discovered
  schemas/
    ros-humble.yaml         # auto-discovered, layered onto inline schema
    project-extras.yaml     # auto-discovered
  profiles/
    base.yaml
    base-schemas/           # NOT discovered (must be named "schemas/")
  profiles/schemas/
    profile-override.yaml   # auto-discovered for profiles in this directory
```

### Single-file vs. multi-file

For small projects, put everything in one file (inline profile + inline schema + defaults). For larger projects, extract profiles and schemas into separate files -- auto-discovery handles the wiring.

## Commands

| Command | Description |
|---|---|
| `init` | Scaffold a product spec (`--name`) or config file (`--config`) |
| `validate` | Validate specs -- auto-discovers product, validates inline schemas and profiles |
| `resolve` | Resolve dependencies -- auto-discovers product, schemas, reads spec defaults |
| `lock` | Alias for `resolve` |
| `build` | Resolve + build debs -- auto-discovers product, schemas, reads spec defaults |
| `publish` | Publish artifacts and create a snapshot |
| `inspect` | Inspect resolved outputs and bundle membership |
| `repo-index` | Generate a repository index from APT and PyPI feeds |
| `prune` | Prune snapshot distributions based on retention policy |

All commands that accept `--product` will auto-discover `product.yaml` in the current directory when the flag is omitted. Run `avular-packages <command> --help` for flag details.

## Schema Resolution

Standard ROS `package.xml` tags like `<depend>`, `<exec_depend>`, and `<build_depend>` declare abstract dependency keys (e.g. `opencv`). Schema mappings resolve these to concrete, typed packages.

There are four ways to provide schemas, listed from lowest to highest precedence:

**1. Inline schema** -- embedded directly in the product or profile spec:

```yaml
schema:
  schema_version: "v1"
  mappings:
    opencv:
      type: apt
      package: libopencv-dev
      version: ">=4.5"
    numpy:
      type: pip
      package: numpy
```

**2. Auto-discovered files** -- any `.yaml`/`.yml` files in a `schemas/` directory next to the product spec or profile files are loaded automatically (sorted alphabetically):

```
my-project/
  product.yaml
  schemas/
    ros-humble.yaml       # loaded automatically
    project-extras.yaml   # loaded automatically, overrides ros-humble per key
```

**3. Spec `schema_files`** -- explicit paths in the spec's `inputs.package_xml.schema_files`

**4. CLI `--schema`** (highest) -- files passed on the command line override everything else:

```bash
avular-packages resolve --schema hotfix-override.yaml
```

Within each layer, later entries override earlier ones per key. The `--schema` flag is available on both `resolve` and `build` commands.

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
