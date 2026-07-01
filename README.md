# q2git

`q2git` is a wasmCloud component that queries remote APIs, transforms results with JQ, and commits them to a git repository.

## Configuration

Configuration is provided at runtime through environment variables — nothing is
embedded in the WASM binary.

| Variable | Required | Purpose |
|---|---|---|
| `Q2GIT_CONFIG` | yes | Full config YAML (schema below) |
| `Q2GIT_GITHUB_TOKEN` | yes | PAT with write access to the destination repo |
| `Q2GIT_SOURCE_USERNAME` | no | Basic-auth username for the source API |
| `Q2GIT_SOURCE_PASSWORD` | no | Basic-auth password for the source API |

On wasmCloud + Kubernetes, `configFrom` (ConfigMap) and `secretFrom` (Secret) on
the component's `localResources.environment` inject these values as env vars.

### Example `Q2GIT_CONFIG`

See [`config.yaml.example`](config.yaml.example):

```yaml
settings:
  write_mode: append   # or "overwrite"

source:
  method: GET
  headers:
    User-Agent: "q2git/1.0"
    Accept: "application/json"

queries:
  - name: open-issues
    description: Top 10 open issues
    url: https://api.github.com/repos/octocat/Hello-World/issues
    query: |
      [.[] | {number, title, state}] | .[0:10]

destination:
  api_url: https://api.github.com
  owner: mihaigalos
  repo: test
  branch: main
  output_path: data.json
  commit_message: Update query results
```

The `destination.token` field is not accepted from YAML — the token
must come from `Q2GIT_GITHUB_TOKEN`.

## Local development

```bash
export Q2GIT_CONFIG="$(cat config.yaml)"
export Q2GIT_GITHUB_TOKEN=github_pat_...
just dev
```

## API

```bash
curl -X POST http://localhost:8000/api/execute
curl -X POST "http://localhost:8000/api/execute?query=power-consumption"
curl -X POST "http://localhost:8000/api/commit"
curl -X POST "http://localhost:8000/api/commit?query=power-consumption"
```

## Build and push

```bash
just build
just push
```
