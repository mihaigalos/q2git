# q2git

`q2git` is a wasmCloud component that queries remote APIs, transforms results with JQ, and commits them to a git repository.

## Configuration Files

Two configuration files are embedded into the WASM binary at build time: **config.yaml** and **secrets.yaml**.

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
curl -X POST http://localhost:8000/api/execute                          # Execute all queries
curl -X POST "http://localhost:8000/api/execute?query=power-consumption" # Execute a specific query by name
curl -X POST "http://localhost:8000/api/execute?commit=true"            # Execute all queries and commit to git
curl -X POST "http://localhost:8000/api/execute?query=power-consumption&commit=true" # Execute one query and commit
```

## Example config

### config.yaml

```yaml
settings:
  write_mode: "append" # "overwrite" or "append"

source:
  method: "GET"
  headers:
    User-Agent: "q2git/1.0"
    Accept: "application/json"

queries:
  - name: "open-issues"
    description: "Top 10 open issues"
    url: "https://api.github.com/repos/octocat/Hello-World/issues"
    query: |
      [.[] | {number: .number, title: .title, state: .state}] | .[0:10]

  - name: "repo-stars"
    description: "Star count for a repository"
    url: "https://api.github.com/repos/octocat/Hello-World"
    query: |
      {name: .full_name, stars: .stargazers_count}

destination:
  api_url: "https://api.github.com"
  owner: "mihaigalos"
  repo: "test"
  branch: "main"
  output_path: "data.json"
  commit_message: "Update query results"
```

### secrets.yaml (do not commit)

```yaml
source:
  username: ""
  password: ""

destination:
  token: "github_pat_xxxxxxxxxxxxxxxxxxxxxx_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```
