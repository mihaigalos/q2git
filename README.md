# q2git

`q2git` is a wasmCloud application which queries a remote api and commits results to git.

`jq` is used internally to mutate results before committing.

## Configuration Files

Two configuration files that are embedded into the WASM binary at build time: **config.yaml** and **secrets.yaml**.

## Usage

```bash
just build
just deploy
```

## Testing

```bash
just test
```

Or manually POST to these endpoints:

```bash
curl -X POST http://localhost:8000/api/execute # Execute query without committing
curl -X POST "http://localhost:8000/api/execute?commit=true" # Execute and commit to git
```

## Example config

### config.yaml

```yaml
source:
  url: "https://api.github.com/repos/octocat/Hello-World/issues"
  method: "GET"
  headers:
    User-Agent: "q2git/1.0"
    Accept: "application/json"

# JQ query to transform the fetched data
query: |
  {
    timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ")),
    results: [.[] | {number: .number, title: .title, state: .state}] | .[0:10]
  }

destination:
  api_url: "https://api.github.com"
  owner: "mihaigalos"
  repo: "test"
  branch: "main"
  output_path: "data_wasmcloud.json"
  commit_message: "Update query results from {{.Timestamp}}"
```

### secrets.yaml (do not commit)

```yaml
source:
  username: ""
  password: ""

destination:
  token: "github_pat_xxxxxxxxxxxxxxxxxxxxxx_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```
