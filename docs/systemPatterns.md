---
title: System Patterns
project: avular-packages
status: Draft
last_updated: 2026-01-27
---

# System Patterns

## 1) Architecture Pattern

The system uses **Hexagonal Architecture (Ports and Adapters)**.

- Domain core holds business logic and policies.
- Ports define interfaces that the core depends on.
- Adapters implement ports to interact with files, repos, and packaging systems.

## 2) Module Layout (Go)

No code is defined here, only package boundaries.

- `cmd/avular-packages`  
  CLI entrypoint.
- `internal/app`  
  Application services orchestrating use-cases.
- `internal/core`  
  Domain services: `SpecCompiler`, `ProductComposer`, `ResolverCore`.
- `internal/ports`  
  Port interfaces used by the core.
- `internal/adapters`  
  Implementations for file I/O, repo publishing, package build, and compatibility outputs.
- `internal/policies`  
  Conflict resolution and packaging mode enforcement.
- `internal/types`  
  Shared domain types (dependency nodes, constraints, lock artifacts).
- `pkg/contract`  
  Public contract types for consumers (optional if needed).

## 3) Port Contracts (High-Level)

### 3.1 Input Ports (Drive the Core)

- `ProductSpecPort`: load and validate product specs.
- `ProfileSpecPort`: load profile specs referenced by products.
- `PackageXmlPort`: discover and parse `package.xml` exports.
- `WorkspacePort`: enumerate workspace packages and metadata.

### 3.2 Output Ports (Driven by the Core)

- `PackageBuildPort`: build deb artifacts for internal and Python deps.
- `RepoSnapshotPort`: publish and promote snapshots in the apt repo (ProGet primary, aptly mirror, file for dev).
- `CompatibilityPort`: emit `get-dependencies` lists and rosdep mappings.
- `PolicyPort`: apply explicit packaging modes and conflict directives.

## 4) Dependency Rules

- Domain core **must not** depend on adapters.
- Ports **must** depend on core types only.
- Adapters **must** implement ports without adding policy.

## 5) Invariants

- Explicit packaging mode per dependency group.
- Fail-closed conflict resolution without directive.
- Deterministic outputs for identical inputs + snapshot state.
