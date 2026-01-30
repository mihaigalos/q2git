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
curl -X POST http://localhost:8000/api/execute # Execute query without committing
curl -X POST "http://localhost:8000/api/execute?commit=true" # Execute and commit to git
```

## Example config

### config.yaml

```yaml
source:
  url: "https://api.github.com/repos/owner/repo/issues"
  method: "GET"
  headers:
    User-Agent: "q2git/1.0"
    Accept: "application/json"

query: |
  .[] | {number: .number, title: .title, state: .state}

git:
  api_url: "https://api.github.com"
  owner: "your-username"
  repo: "your-repo"
  branch: "main"
  output_path: "query-results/data.json"
  commit_message: "Update query results from {{.Timestamp}}"
```

### secrets.yaml (do not commit)

```yaml
source:
  username: ""
  password: ""

git:
  token: "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```