---
title: Written Specification
project: avular-packages
status: Draft
last_updated: 2026-01-27
---

# Written Specification

## 1) Scope
This specification defines a single Ubuntu-focused dependency management system that produces deterministic, rollbackable dependency sets using **dpkg/deb** as the runtime format and a **ProGet-hosted Debian feed** as the primary runtime channel, with snapshot-style distributions and weekly upstream mirroring via aptly.

## 2) Normative Language
The terms **MUST**, **MUST NOT**, **SHOULD**, and **MAY** are used as defined in RFC 2119.

## 3) Functional Requirements
- The system **MUST** accept product-level specs and compose them in a deterministic order.
- The system **MUST** ingest ROS `package.xml` export tags (`debian_depend`, `pip_depend`) as dependency inputs.
- The resolver **MUST** produce an exact apt lockfile (package=version) per product spec.
- The resolver **MUST** emit a snapshot identifier that uniquely references the published repository state.
- The system **MUST** package Python dependencies into debs for runtime consumption.
- The system **MUST** publish artifacts to a ProGet Debian feed with snapshot-style distributions and immutable snapshots.
- The system **MUST** implement hexagonal architecture with explicit ports and adapters.

## 4) Non-Functional Requirements
- The system **MUST** support rollback through snapshot pointer changes.
- The system **MUST** provide verifiable provenance and signed repository metadata.
- The system **SHOULD** minimize developer friction and preserve existing workflows during transition.
- The system **MUST** be deterministic and reproducible across CI and dev environments.

## 5) Dependency Resolution Rules
### 5.1 Priority Order
1) Product spec explicit pins  
2) Product profile overrides  
3) `package.xml` constraints  
4) Resolver defaults

### 5.2 Conflict Handling
- If constraints intersect, the resolver **MUST** choose the best compatible version.
- If constraints do not intersect, the resolver **MUST** fail closed unless a resolution directive is present.
- A resolution directive **MUST** include: `action`, `reason`, `owner`. `expires_at` is optional.

## 6) Packaging Modes (Explicit)
Every dependency group **MUST** explicitly declare one packaging mode:
- **individual**: one deb per dependency
- **meta-bundle**: one deb that depends on individual debs
- **fat-bundle**: one deb that contains a vendored environment

No default mode is allowed. Validation **MUST** fail if a mode is missing.

## 7) Repository and Release Model
- The publish layer **MUST** create immutable snapshot distributions in ProGet.
- Snapshots **MUST** be signed and versioned via ProGet feed signing keys.
- The system **MUST** support dev → staging → prod promotion with no rebuilds via distribution updates.
- Installers **MUST** reference snapshot distributions or promoted channels, not mutable ad-hoc sources.

## 8) Python Dependency Channel Policy
- The pip index **MUST NOT** be used as a runtime channel.
- The pip index **MAY** be used as a build input to create debs.
- All runtime Python dependencies **MUST** be installed from the apt repo.

## 9) Compatibility Requirements
- The system **SHOULD** provide a compatibility wrapper for `get-dependencies` during migration.
- The system **MAY** export rosdep-style mappings for legacy workflows.

## 10) Security and Integrity
- Repository metadata **MUST** be signed.
- Snapshots **MUST** be immutable once published.
- The system **SHOULD** generate SBOMs in SPDX JSON and store them alongside snapshots.

**Note:** When using ProGet, signing keys are managed at the feed level; `signing_key` is applied by the aptly backend.

