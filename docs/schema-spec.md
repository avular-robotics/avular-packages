---
title: Schema Specification
project: avular-packages
status: Draft
last_updated: 2026-01-27
---

# Schema Specification

## 1) Overview

This schema defines **product-level, composable specifications** for dependency resolution and packaging. Profiles are reusable components (base, platform, environment) that can be composed into product specs. Every packaging decision is explicit.

## 2) Core Concepts

- **Profile:** A reusable block of dependency policy and packaging rules.
- **Product Spec:** A composition of profiles plus product-specific overrides.
- **Resolution Directive:** A required, explicit decision for any dependency conflict.
- **Packaging Mode:** Must be declared per dependency group (individual, meta-bundle, fat-bundle).

## 3) File Types

### 3.1 Profile Spec

- Intended for reuse across products (base OS, platform, environment).

### 3.2 Product Spec

- Composes profiles and sets final policy.
- This is the source of truth for release and snapshots.

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

## 5) Example: Profile Spec

```yaml
api_version: "v1"
kind: "profile"
metadata:
  name: "ubuntu-jammy-base"
  version: "2026.01"
  owners: ["platform"]
inputs:
  package_xml:
    enabled: true
    tags: ["debian_depend", "pip_depend"]
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

## 6) Example: Product Spec

```yaml
api_version: "v1"
kind: "product"
metadata:
  name: "origin-robot"
  version: "2026.01.27"
  owners: ["origin-team"]
compose:
  - name: "ubuntu-jammy-base"
    version: "2026.01"
    source: "git"
    path: "profiles/ubuntu-jammy-base.yaml"
inputs:
  package_xml:
    enabled: true
    tags: ["debian_depend", "pip_depend"]
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

## 7) Validation Rules

- `packaging.groups[].mode` is required and cannot be defaulted.
- `compose` is required for product specs.
- Any conflict must have a matching `resolutions` entry.
- Targets must be Ubuntu releases only.
