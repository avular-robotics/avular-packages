---
title: Justfile Contract
project: avular-packages
status: Draft
last_updated: 2026-01-27
---

# Justfile Contract

## 1) Purpose
Define a stable, human-friendly task interface for local development and CI. This document describes the **expected tasks**, **inputs**, and **outputs**, without defining implementation details.

## 2) Task Taxonomy (Required)
### 2.1 Resolution
- `resolve`: validate specs and compute dependency graph.
  - Inputs: product spec, profile specs, workspace paths
  - Outputs: resolution report, bundle manifest

### 2.2 Locking
- `lock`: produce apt lockfile and snapshot intent.
  - Inputs: resolved graph, target Ubuntu release
  - Outputs: `apt.lock`, `snapshot.intent`

### 2.3 Build
- `build`: build deb artifacts for internal code and Python deps.
  - Inputs: lock outputs, build inputs (wheels/sdists)
  - Outputs: deb artifacts

### 2.4 Publish
- `publish`: publish debs and create snapshot.
  - Inputs: deb artifacts, signing key, repo backend (ProGet/aptly/file)
  - Outputs: published snapshot (ProGet primary, aptly mirror, file dev)

### 2.5 Validate
- `validate`: schema validation and policy checks.
  - Inputs: specs, profiles
  - Outputs: validation report

### 2.6 Inspect
- `inspect`: show resolved packages, pins, and bundle membership.
  - Inputs: resolution outputs
  - Outputs: human-readable summary

### 2.7 Clean
- `clean`: remove build artifacts and caches.

## 3) Behavioral Guarantees
- Tasks are **deterministic** given identical inputs and snapshot state.
- Validation **fails closed** on conflicts without resolution directives.
- No task modifies runtime channels outside the publish workflow.

## 4) Environment Requirements
- Go toolchain and required system packages (see devcontainer spec).
- Access to signing keys for publish tasks (CI only).

