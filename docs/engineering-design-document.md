---
title: Engineering Design Document
project: avular-packages
status: Draft
last_updated: 2026-01-27
---

# Engineering Design Document

## 1) Executive Summary
This document defines a single, production-grade dependency management system for Ubuntu that replaces the current split between rosdep mappings, ad-hoc pip installs, and multiple internal tools. The system uses **dpkg/deb as the single runtime package format** and a **ProGet-hosted Debian feed** as the primary runtime channel, with **aptly used only as a weekly mirror** for upstream history. Python dependencies are packaged as debs, and the pip index is treated as a build input only. The system is **product-spec driven**, **composable**, and uses **explicit packaging decisions** with **fail-closed conflict resolution**.

## 2) Context and Problem Statement
Today, dependency resolution and packaging are split across:
- `get-dependencies` for `package.xml` tags.
- rosdep mappings in a separate repo.
- `avular-dep` and `tue-env` for installation workflows.
- CI scripts that install and lock dependencies in different ways.

This creates friction in version pinning, rollback, and reproducibility. It also maintains two runtime channels (deb and pip), which increases drift and operational complexity.

## 3) Goals
- Single runtime channel for all dependencies.
- Deterministic, version-controlled dependency sets with pinning and rollback.
- Declarative, product-level specs with composition.
- Explicit packaging choices (individual, meta-bundle, fat-bundle) with no defaults.
- Low friction for developers and CI.
- Ubuntu-only support, but robust production-grade release workflows.

## 4) Non-Goals
- Supporting non-Ubuntu OSes at runtime.
- Replacing dpkg/deb as the runtime format.
- Mandating new developer tools that replace existing ROS workflows.
- Removing `package.xml` tags as a dependency source.

## 5) Requirements
### Functional
- MUST parse dependencies from `package.xml` export tags.
- MUST resolve apt + Python dependencies into a unified dependency graph.
- MUST produce lockfiles and snapshot metadata per product spec.
- MUST build deb packages for Python dependencies.
- MUST publish to a ProGet Debian feed with snapshot-style distribution naming and rollback support.

### Non-Functional
- Deterministic resolution and reproducible install sets.
- Strong audit trail and explicit conflict resolution.
- Low-friction UX for developers and CI.

## 6) Key Decisions (Locked)
- **Runtime format:** dpkg/deb.
- **Runtime channel:** ProGet Debian feed (primary) + aptly weekly mirror (upstream-only).
- **Python deps:** packaged as debs (pip index used only as build input).
- **Spec scope:** product-level with explicit composition.
- **Packaging mode:** explicit per dependency group, no default.
- **Conflict policy:** best-compatible selection; fail closed if conflict without explicit resolution directive.

## 7) Proposed Architecture
### 7.1 Component Overview
- **Spec Layer:** Product specs and profiles, explicitly composed.
- **Resolver:** Builds the unified dependency graph and outputs lockfiles.
- **Build System:** Builds debs (internal + Python).
- **Publish System:** ProGet Debian feed with snapshot-style distributions, signing, and promotion.
- **Adapters:** Compatibility wrappers for `get-dependencies` and rosdep export.

### 7.2 Data Flow
1) Product spec composes profiles and pulls `package.xml` exports.
2) Resolver computes an exact dependency graph and versions.
3) Build layer produces deb artifacts.
4) Publish layer creates snapshot and promotes it.
5) Installers and CI consume snapshots and lockfiles.

### 7.3 Hexagonal Architecture (Ports and Adapters)
Core services are isolated behind ports, with adapters providing I/O. The domain core contains no infrastructure dependencies.

**Domain core services**
- `SpecCompiler`: validates and normalizes product specs.
- `ProductComposer`: composes profile layers into a single product view.
- `ResolverCore`: resolves dependencies and conflicts into lock outputs.

**Primary ports (driven by the core)**
- `RepoSnapshotPort`: publish and promote snapshots.
- `PackageBuildPort`: build deb artifacts (including Python debs).
- `PolicyPort`: packaging decisions and conflict directives.

**Secondary ports (driving the core)**
- `ProductSpecPort`: read specs from files.
- `PackageXmlPort`: read and parse `package.xml` exports.
- `WorkspacePort`: enumerate workspace packages and metadata.

**Adapters**
- CLI adapter (cobra) for user commands.
- File adapter for specs and workspace discovery.
- Repository adapter (apt snapshot manager).
- Python packaging adapter (wheel/sdist to deb).

## 8) Packaging Strategy
### 8.1 Packaging Modes (Explicit)
- **Individual debs:** One deb per Python dependency.
- **Meta-bundle debs:** A deb that depends on individual debs.
- **Fat-bundle debs:** Vendored env packaged in one deb.

### 8.2 Policy
Every dependency group must declare its packaging mode explicitly. No defaults.

## 9) Tech Stack (Go)
- **CLI:** `cobra`
- **Config:** `viper` (file + env precedence)
- **Logging:** `zerolog` (structured, high-performance)
- **Errors:** `errbuilder-go`
- **Assertions:** `assert-lib` (source-level checks)
- **Testing:** `testify`

## 10) Conflict Resolution Strategy
### 9.1 Priority Order
1) Product spec explicit pins
2) Product profile overrides
3) `package.xml` constraints
4) Resolver defaults

### 9.2 Behavior
- If constraints intersect: select best compatible version.
- If constraints conflict: fail closed unless an explicit resolution directive is present.

## 11) Repository Strategy
Use a ProGet Debian feed as the primary runtime repository, with:
- Signed Release metadata managed by ProGet feed keys.
- Snapshot-style distributions using `snapshot_prefix` + `snapshot_id`.
- Old snapshot distributions pruned by retention policy.
- Promotion workflow (dev → staging → prod) via distribution updates.
- Snapshot IDs referenced in `snapshot.intent` and product specs.
- `signing_key` in specs is used for aptly; ProGet signing is configured at the feed level.

Use aptly only as a **weekly upstream mirror** (not a primary publish backend).

## 12) Developer Ergonomics
- Developers keep using `package.xml`.
- Compatibility CLI provides `get-dependencies` output for a transition period.
- Product specs are owned by platform/robot/product teams.
- Devcontainer provides the full toolchain (Go + packaging tooling).
- `just` defines standard workflows (resolve, lock, build, publish, validate).

## 13) Migration Plan (High-Level)
1) Introduce resolver and lockfiles in CI for a pilot product.
2) Switch devcontainers to install from snapshots.
3) Package Python dependencies as debs, publish to apt repo.
4) Deprecate separate pip channel and rosdep mapping repo.
5) Retire old tooling once compatible replacements are stable.

## 14) Risks and Mitigations
- **Risk:** Conflicting constraints across profiles.
  - **Mitigation:** explicit resolution directives and fail-closed.
- **Risk:** Migration breaks CI tooling.
  - **Mitigation:** compatibility wrappers for `get-dependencies`.
- **Risk:** Python packaging overhead.
  - **Mitigation:** automated build/publish pipeline with caching.

## 15) Open Questions
- None. All major decisions have been locked by stakeholder input.

## 16) Long-Term Alternatives
- See `docs/alternatives-nix-conda.md` for a Nix/Conda-style design note and decision criteria.

## 17) Implementation Status
- Resolver, spec compiler, composition, and file adapters are implemented.
- Build supports Python deb packaging (individual/meta/fat) and optional internal deb builds.
- Publish supports file backend (dev), ProGet publishing (primary), and aptly-backed snapshot publishing (mirror/ops).
- Compatibility outputs and SBOM generation are implemented as optional steps.
