---
title: Tech Stack Decisions
project: avular-packages
status: Draft
last_updated: 2026-01-27
---

# Tech Stack Decisions

## 1) Go Libraries (Locked)

- **CLI:** `github.com/spf13/cobra`
- **Config:** `github.com/spf13/viper`
- **Logging:** `github.com/rs/zerolog`
- **Errors:** `github.com/ZanzyTHEbar/errbuilder-go`
- **Assertions:** `github.com/ZanzyTHEbar/assert-lib`
- **Testing:** `github.com/stretchr/testify`
- **UUIDs:** `github.com/google/uuid`
- **YAML parsing:** `gopkg.in/yaml.v3`

## 2) Conventions

- Single Go module at repo root.
- Module path: `avular-packages`.
- All core interfaces in `internal/ports`.
- Domain services in `internal/core`.
- Adapters in `internal/adapters`.
- Shared types in `internal/types`.

## 3) System Dependencies (Build/Publish)

- **Deb packaging:** `dpkg-dev`, `debhelper`, `lintian`
- **Tooling:** `build-essential`, `git`, `curl`, `gnupg`
- **Repo management:** ProGet (primary Debian feed) and `aptly` (weekly mirror)
- **Python build inputs:** `python3`, `python3-venv`, `python3-pip`, `pip-tools`, `wheel`

## 4) OS Targets

- Ubuntu LTS matrix only: `22.04`, `24.04`.