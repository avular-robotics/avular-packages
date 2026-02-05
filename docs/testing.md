# Testing

## Offline e2e fixtures

The integration test `internal/integration/proget_upload_testcontainers_test.go` runs fully offline by standing up local fixtures for all external dependencies:

- **ProGet mock**: a testcontainers Python HTTP server that records uploads and exposes `/requests`.
- **APT fixture**: a testcontainers Python HTTP server that serves a minimal `Packages` file under `dists/dev/main/binary-amd64/`.
- **PIP fixture**: a local `httptest` server that builds a wheel on the fly and serves a Simple Index at `/simple/`.

The test uses these endpoints to run repo-index → resolve → build → publish and then verifies the upload and promotion paths.

### Run the test

```
go test ./internal/integration -tags=integration -run TestE2EProGetPublishWithTestcontainers
```

### Extend the fixtures

Update these constants in the test file:

- `aptPackageName`, `aptPackageVersion`
- `pipPackageName`, `pipPackageVersion`

Then adjust the spec inputs in the same file:

- `testPackageXML` for PIP deps
- `buildProductSpec()` for manual APT deps

To add more APT packages, add additional `Package:` blocks in `artifactServerScript` (the `Packages` file).

To add more PIP packages, build additional wheels in `startLocalPipIndex()` and add them to the Simple Index HTML.
