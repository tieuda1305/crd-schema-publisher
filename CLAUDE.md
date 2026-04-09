# CLAUDE.md - Project Context for crd-schema-publisher

## What This Is

A statically-compiled Go binary that extracts CRD JSON schemas from a Kubernetes cluster and publishes them to a Cloudflare Pages website. Runs as an hourly CronJob in a K3s cluster. Deployed in a nonroot distroless container.

## Repository Layout

```text
cmd/            Entrypoint and subcommand dispatch (run/extract/upload)
converter/      OpenAPI v3 -> JSON Schema transforms (ported from openapi2jsonschema.py)
extractor/      client-go CRD listing, schema extraction, file writing
index/          Static index.html generation
publisher/      Cloudflare Pages direct upload API client + BLAKE3 hashing
```

## Build & Test

```bash
# Run all tests
go test ./...

# Vet
go vet ./...

# Cross-compile (matches CI targets)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o crd-schema-publisher ./cmd/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o crd-schema-publisher ./cmd/

# Local extract (requires kubeconfig)
KUBECTL_CONTEXT=my-context OUTPUT_DIR=./output go run ./cmd/ extract
```

## Architecture

### Subcommands

- `run` (default) — extract + upload
- `extract` — extract CRDs and write JSON schemas + index.html to OUTPUT_DIR
- `upload` — upload OUTPUT_DIR contents to Cloudflare Pages

### Configuration (env vars)

| Var | Required | Default | Purpose |
|-----|----------|---------|---------|
| `CLOUDFLARE_API_TOKEN` | Yes (run/upload) | — | CF API token |
| `CLOUDFLARE_ACCOUNT_ID` | Yes (run/upload) | — | CF account ID |
| `CF_PAGES_PROJECT` | No | `kubernetes-schemas` | CF Pages project name |
| `OUTPUT_DIR` | No | `/output` | Schema output directory |
| `KUBECTL_CONTEXT` | No | — | K8s context (local dev only) |

### Key Design Decisions

- **No CGO.** Binary is statically linked. BLAKE3 uses `github.com/zeebo/blake3` (pure Go).
- **Cloudflare Pages direct upload API** is undocumented. Implementation reverse-engineered from wrangler source (`cloudflare/workers-sdk`). The upload flow uses JWT auth for asset operations and API token auth for deployment creation. See `publisher/publisher.go` for the full 6-step flow.
- **BLAKE3 file hashing** exactly matches wrangler's `hashFile`: `hex(blake3(base64(content) + extension))[0:32]`. Do not change this algorithm without verifying against wrangler source.
- **OpenAPI v3 to JSON Schema conversion** is a faithful port of `openapi2jsonschema.py` from datreeio/CRDs-catalog. Three transforms applied in order: additionalProperties, replaceIntOrString, allowNullOptionalFields.
- **Output format** produces two directory structures: `<group>/<kind>_<version>.json` (primary) and `master-standalone/<group>-<kind>-stable-<version>.json` (kubeval compatibility).
- **Concurrency**: extractor uses 10 goroutines (buffered channel semaphore), publisher uses 3 concurrent upload workers. These match the original tools' behavior.

### Dependencies (direct only)

| Dependency | Purpose |
|-----------|---------|
| `k8s.io/client-go` | Kubernetes API access |
| `k8s.io/apiextensions-apiserver` | CRD typed client |
| `github.com/zeebo/blake3` | BLAKE3 hashing (pure Go) |

No other external dependencies. Standard library for HTTP, JSON, HTML templates, MIME types.

## CI/CD

- **GitHub Actions** (`.github/workflows/build.yaml`): builds multi-arch image (amd64 + arm64) on push to main, pushes to `ghcr.io/sholdee/crd-schema-publisher`
- **Tags**: date-based (`YYYY-MM-DD-HHMMSS` UTC) + `latest`
- **Container**: `gcr.io/distroless/static:nonroot` runtime base

## Companion Repo

Kubernetes manifests live in the `home-ops` repo under `apps/kubernetes-schemas/`. That repo has its own CLAUDE.md with cluster conventions, RBAC, ExternalSecret, and CronJob configuration.

## Common Mistakes to Avoid

- Changing the BLAKE3 hash algorithm without verifying against wrangler — uploads will fail with hash mismatches
- Adding CGO dependencies — breaks static compilation and distroless compatibility
- Modifying the CF Pages upload flow without checking current wrangler source — the API is undocumented and may change
- Forgetting to update both output directory formats (primary + master-standalone) when changing schema file naming
