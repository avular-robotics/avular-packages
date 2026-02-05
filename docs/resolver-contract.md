---
title: Resolver Contract Specification
project: avular-packages
status: Draft
last_updated: 2026-01-27
---

# Resolver Contract Specification

## 1) Purpose

Defines the contract between the dependency resolver and its consumers (CI, devcontainers, publish pipeline). This contract guarantees deterministic outputs, explicit conflict handling, and reproducible installations.

## 2) Inputs

The resolver consumes:

- **Product spec** (required): composable product definition.
- **Profile specs** (required if referenced): base/platform/environment definitions.
- **Workspace sources** (optional): paths containing `package.xml` files.
- **Overrides** (optional): manual dependency pins or replacements.
- **Target OS**: Ubuntu release (required, single value).

## 3) Outputs

The resolver produces:

- **apt.lock**: exact list of `package=version` entries.
- **bundle.manifest**: packaging mode and grouping for dependencies.
- **snapshot.intent**: metadata describing the intended repo snapshot (name, prefix, channel, signing key).
- **resolution.report**: decisions, conflicts, and applied directives.

## 4) Determinism Guarantees

- Given the same inputs and repository snapshot state, outputs **MUST** be identical.
- The resolver **MUST NOT** call external mutable channels during resolution.
- The resolver **MUST** require explicit resolution directives for conflicts.

## 5) Conflict Rules (Executable Contract)

- If constraints intersect, select highest compatible version within snapshot.
- If constraints do not intersect, **fail closed** unless a resolution directive exists.
- A resolution directive **MUST** include: `action`, `reason`, `owner`.

## 6) Packaging Mode Enforcement

Every dependency group **MUST** declare a packaging mode:

- `individual`  
- `meta-bundle`  
- `fat-bundle`

Resolver **MUST** fail validation if any group lacks a mode.

## 7) Error Conditions

The resolver **MUST** fail closed on:

- Missing or invalid product spec.
- Missing referenced profile spec.
- Unsupported Ubuntu release.
- Missing packaging mode for any dependency group.
- Conflicting constraints without resolution directive.
- Unresolvable dependency (no compatible version in snapshot).

## 8) Status and Exit Codes

- **0**: success, all outputs produced.
- **2**: validation failure (schema, composition, missing packaging mode).
- **3**: conflict without resolution directive.
- **4**: dependency resolution failure (no compatible version).
- **5**: environment failure (missing inputs, unsupported OS).

## 9) Output Formats (Minimal)

### 9.1 apt.lock

- One entry per line: `package=version`
- Sorted lexicographically by package name.

### 9.2 bundle.manifest

- Records packaging mode and group membership.
- Fields: `group`, `mode`, `package`, `version`.

### 9.3 snapshot.intent

- Fields: `repository`, `channel`, `snapshot_prefix`, `snapshot_id`, `created_at`, `signing_key`.

### 9.4 resolution.report

- Lists conflicts, applied directives, and final decisions.
- Fields: `dependency`, `action`, `value`, `reason`, `owner`, `expires_at`.

## 10) Idempotency

- Resolver output **MUST** be identical on repeated runs with the same inputs and snapshot state.

## 11) Compatibility Outputs

If enabled, the resolver **MAY** emit:

- `get-dependencies` compatible lists (apt and pip).
- rosdep-style mappings for legacy workflows.
