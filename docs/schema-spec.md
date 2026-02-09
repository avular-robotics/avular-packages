---
title: Schema Specification
project: avular-packages
status: Active
last_updated: 2026-02-09
---

# Schema Specification

## 1) Overview

This schema defines **product-level, composable specifications** for dependency resolution and packaging. Profiles are reusable components (base, platform, environment) that can be composed into product specs. Every packaging decision is explicit.

The system supports two complementary dependency input methods:

1. **Export tags** (`<debian_depend>`, `<pip_depend>`) -- typed, concrete dependencies declared directly in `package.xml` `<export>` sections.
2. **Schema-resolved ROS tags** (`<depend>`, `<exec_depend>`, `<build_depend>`, etc.) -- abstract dependency keys from standard ROS `package.xml` tags, mapped to concrete typed packages through a `schema.yaml` file.

## 2) Core Concepts

- **Profile:** A reusable block of dependency policy and packaging rules.
- **Product Spec:** A composition of profiles plus product-specific overrides.
- **Resolution Directive:** A required, explicit decision for any dependency conflict.
- **Packaging Mode:** Must be declared per dependency group (individual, meta-bundle, fat-bundle).
- **Schema Mapping:** A versioned, layered file that maps abstract ROS dependency keys to concrete typed installable packages.

## 3) File Types

### 3.1 Profile Spec

- Intended for reuse across products (base OS, platform, environment).

### 3.2 Product Spec

- Composes profiles and sets final policy.
- This is the source of truth for release and snapshots.

### 3.3 Schema Mapping (`schema.yaml`)

- Maps abstract dependency keys to concrete typed packages.
- Layered: workspace -> profile -> product (last loaded wins per key).
- Referenced from `inputs.package_xml.schema_files` or via `--schema` CLI flag.

## 4) Schema (Logical)

### 4.1 Common Fields

- `api_version`: string, required.
- `kind`: enum: `profile` | `product`, required.
- `metadata`: object, required.
  - `name`: string, required.
  - `version`: string, required (semantic or date-based).
  - `owners`: list of strings, required.
  - `description`: string, optional.

### 4.2 Composition

- `compose`: ordered list of profile references (product only).
  - `name`: string, required.
  - `version`: string, required.
  - `source`: enum: `git` | `local`, required.
  - `path`: string, required for `local`.

### 4.3 Dependency Inputs

- `inputs.package_xml`:
  - `enabled`: bool, required.
  - `tags`: list of strings, required (e.g., `debian_depend`, `pip_depend`).
  - `include_src`: bool, optional.
  - `prefix`: string, optional (deb package prefix for workspace filtering).
  - `schema_files`: list of paths to `schema.yaml` files (optional). Loaded in order; later files override earlier ones per key.

- `inputs.manual`:
  - `apt`: list of package constraints.
  - `python`: list of package constraints.

### 4.4 Packaging Rules (Explicit)

- `packaging.groups`: list of dependency group definitions.
  - `name`: string, required.
  - `mode`: enum `individual` | `meta-bundle` | `fat-bundle`, required.
  - `scope`: enum `runtime` | `dev` | `test` | `doc`, required.
  - `matches`: list of match rules (by name, tag, namespace).
  - `targets`: list of Ubuntu releases, required.
  - `pins`: list of version constraints (optional).

### 4.5 Conflict Resolution

- `resolutions`: list of directives.
  - `dependency`: string, required.
  - `action`: enum `force` | `relax` | `replace` | `block`, required.
  - `value`: string (required for `force` and `replace`).
  - `reason`: string, required.
  - `owner`: string, required.
  - `expires_at`: date string, optional.

### 4.6 Publishing

- `publish.repository`:
  - `name`: string, required.
  - `channel`: string, required (recommended: `dev` | `staging` | `prod`).
  - `snapshot_prefix`: string, required.
  - `signing_key`: string, required.

## 5) Schema Mapping Specification

### 5.1 File Format

```yaml
schema_version: "v1"
target: "ubuntu-22.04"          # optional platform constraint
mappings:
  <abstract-key>:
    type: "apt" | "pip"          # required
    package: "<concrete-name>"   # required
    version: "<constraint>"      # optional (e.g., ">=1.0,<2.0")
```

### 5.2 Fields

- `schema_version`: string, required. Must be `"v1"`.
- `target`: string, optional. Identifies the target OS/platform.
- `mappings`: map, required. Keys are abstract dependency names (as found in ROS `package.xml` tags).
  - `type`: enum `"apt"` | `"pip"`, required.
  - `package`: string, required. Concrete package name in the target ecosystem.
  - `version`: string, optional. Supports standard version constraint operators: `>=`, `<=`, `==`, `!=`, `~=`, `>`, `<`, `=`, and compound constraints with commas (e.g., `">=1.0,<2.0"`).

### 5.3 Layering

Multiple schema files can be loaded in sequence. Later files override earlier ones on a per-key basis. This enables a layered precedence model:

```
workspace schema  (loaded first)
  ↓ overridden by
profile schema
  ↓ overridden by
product schema
  ↓ overridden by
CLI --schema flag  (loaded last = highest precedence)
```

### 5.4 Resolution Flow

```
package.xml               schema.yaml             resolver
┌─────────────┐          ┌──────────────┐        ┌────────────┐
│ <depend>    │──key──>  │ mappings:    │─typed─>│ Dependency  │
│  rclcpp     │          │   rclcpp:    │  dep   │ Name: ros-  │
│ </depend>   │          │     type: apt│        │   humble-   │
│             │          │     package: │        │   rclcpp    │
│ <exec_dep>  │──key──>  │       ros-.. │        │ Type: apt   │
│  numpy      │          │   numpy:     │        │             │
│ </exec_dep> │          │     type: pip│        │ ...         │
└─────────────┘          │     package: │        └────────────┘
                         │       numpy  │              │
                         │     version: │              ▼
                         │       >=1.26 │        existing resolve
                         └──────────────┘        → lock → build
```

### 5.5 Supported ROS Tags

The following standard ROS package.xml tags are parsed as abstract keys:

| XML Tag | Scope | Description |
|---------|-------|-------------|
| `<depend>` | `all` | Combined build + exec dependency |
| `<exec_depend>` | `exec` | Runtime dependency |
| `<build_depend>` | `build` | Build-time dependency |
| `<build_export_depend>` | `build_exec` | Exported build dependency |
| `<run_depend>` | `exec` | Runtime dependency (deprecated alias) |
| `<test_depend>` | `test` | Test dependency |

Unknown keys (no entry in the schema) are logged as warnings and skipped. Workspace-internal package names are automatically filtered.

## 6) Example: Schema Mapping

```yaml
schema_version: "v1"
target: "ubuntu-22.04"
mappings:
  # ROS packages
  rclcpp:
    type: apt
    package: ros-humble-rclcpp
  std_msgs:
    type: apt
    package: ros-humble-std-msgs
  ament_cmake:
    type: apt
    package: ros-humble-ament-cmake
  rosidl_default_generators:
    type: apt
    package: ros-humble-rosidl-default-generators

  # System libraries
  fmt:
    type: apt
    package: libfmt-dev
    version: ">=9.1.0"
  opencv:
    type: apt
    package: libopencv-dev
    version: ">=4.5"

  # Python packages
  numpy:
    type: pip
    package: numpy
    version: ">=1.26,<2.0"
  flask:
    type: pip
    package: flask
    version: ">=3.0"
```

## 7) Example: Profile Spec (with Schema)

```yaml
api_version: "v1"
kind: "profile"
metadata:
  name: "ubuntu-jammy-base"
  version: "2026.02"
  owners: ["platform"]
inputs:
  package_xml:
    enabled: true
    tags: ["debian_depend", "pip_depend"]
    schema_files:
      - "schemas/ros-humble-jammy.yaml"
      - "schemas/platform-base.yaml"
packaging:
  groups:
    - name: "python-core-individual"
      mode: "individual"
      scope: "runtime"
      matches: ["python:*", "pip:*"]
      targets: ["ubuntu-22.04"]
publish:
  repository:
    name: "avular"
    channel: "dev"
    snapshot_prefix: "base"
    signing_key: "avular-release"
```

## 8) Example: Product Spec

```yaml
api_version: "v1"
kind: "product"
metadata:
  name: "origin-robot"
  version: "2026.02.09"
  owners: ["origin-team"]
compose:
  - name: "ubuntu-jammy-base"
    version: "2026.02"
    source: "git"
    path: "profiles/ubuntu-jammy-base.yaml"
inputs:
  package_xml:
    enabled: true
    tags: ["debian_depend", "pip_depend"]
    schema_files:
      - "schemas/origin-overrides.yaml"
  manual:
    apt:
      - "ros-humble-rmw-cyclonedds-cpp=1.0.2"
packaging:
  groups:
    - name: "origin-python-meta"
      mode: "meta-bundle"
      scope: "runtime"
      matches: ["pip:flask", "pip:pandas"]
      targets: ["ubuntu-22.04"]
resolutions:
  - dependency: "pip:pandas"
    action: "force"
    value: "2.1.4"
    reason: "Compat with bt_nodes_reporting"
    owner: "origin-team"
publish:
  repository:
    name: "avular"
    channel: "staging"
    snapshot_prefix: "origin"
    signing_key: "avular-release"
```

## 9) Validation Rules

- `packaging.groups[].mode` is required and cannot be defaulted.
- `compose` is required for product specs.
- Any conflict must have a matching `resolutions` entry.
- Targets must be Ubuntu releases only.
- Schema files must have `schema_version: "v1"`.
- Schema mappings must have a valid `type` (`apt` or `pip`) and a non-empty `package`.

## 10) CLI Flags

Schema files can be provided via:

1. **Spec file** -- `inputs.package_xml.schema_files` in profile or product YAML.
2. **CLI flag** -- `--schema <path>` (repeatable). CLI schemas are appended after spec schemas, giving them highest precedence.

```bash
# Schema from spec only
avular-packages resolve --product product.yaml --repo-index repo.yaml \
    --target-ubuntu 22.04 --output out

# Schema from CLI (overrides spec schemas)
avular-packages resolve --product product.yaml --repo-index repo.yaml \
    --target-ubuntu 22.04 --output out \
    --schema schemas/base.yaml --schema schemas/override.yaml
```

## 11) Migration Path

### From export-only to schema-resolved

1. **Keep existing `<debian_depend>` and `<pip_depend>` tags** -- they continue to work unchanged.
2. **Create a `schema.yaml`** mapping your workspace's abstract ROS keys to concrete packages.
3. **Reference the schema** in your profile's `inputs.package_xml.schema_files`.
4. **Gradually adopt standard ROS tags** (`<depend>`, `<exec_depend>`) in new package.xml files while keeping export tags in existing ones.
5. Both input methods coexist and produce dependencies that merge into the same resolution pipeline.

### Precedence Order

Dependencies from all sources merge into a single resolution graph:

```
1. manual.apt / manual.python        (highest explicit priority)
2. resolutions directives             (force/replace/relax/block)
3. export tags (debian_depend, pip_depend)
4. schema-resolved ROS tags
5. repo-index available versions
6. SAT solver (if --apt-sat-solver)   (transitive closure)
```
