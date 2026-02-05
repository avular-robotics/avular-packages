---
title: Devcontainer Specification
project: avular-packages
status: Draft
last_updated: 2026-01-27
---

# Devcontainer Specification

## 1) Purpose
Provide a consistent development environment for avular-packages across the Ubuntu LTS matrix.

## 2) Target OS Matrix
- Ubuntu LTS releases only (e.g., 22.04 and 24.04).
- The devcontainer **must** be configurable per LTS version.

## 3) Required Tooling
### 3.1 Go Toolchain
- Go toolchain matching project standard (version pinned in the devcontainer definition).

### 3.2 Build and Packaging
- `dpkg-dev`, `debhelper`, `lintian`
- `build-essential`, `git`, `curl`, `gnupg`
- `aptly` for weekly mirror workflows (ProGet remains the primary backend)

### 3.3 Python Build Inputs
- `python3`, `python3-venv`, `python3-pip`, `pip-tools`, `wheel`

### 3.4 Task Runner
- `just` installed and available on PATH

## 4) Devcontainer Features (Conceptual)
- Base image per Ubuntu LTS.
- Volume mounts for repository and build cache.
- Optional SSH agent forwarding for private dependencies.
- Optional cache for Go modules and build artifacts.

## 5) Non-Functional Requirements
- Fast startup (cache dependencies where possible).
- Reproducible toolchain versions.
- Clear separation between dev and publish credentials.

