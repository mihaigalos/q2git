# Justfile for wasmCloud hello-world application

# Set PATH to use the native-tls version of wash
export PATH := env_var('HOME') + "/.cargo/bin:" + env_var('PATH')

registry := "docker.io/mihaigalos"
tag      := `grep '^version' wasmcloud.toml | sed 's/version = "//;s/"//'`
app      := "q2git"

@_default:
    just --list

# Full development cycle: build, deploy, and test
dev: up deploy
    @echo "✅ Application deployed! Testing..."
    @sleep 3
    @just test

# Test the deployed application
test: deploy
    #!/bin/bash
    printf "\n⏳ Waiting for component to be ready (retrying on connection refused)...\n"
    until http_code=$(curl -s -o /tmp/q2git_out -w "%{http_code}" -X POST http://localhost:8000/api/execute 2>/dev/null) && [ "$http_code" = "200" ]; do
        # 000 = connection refused (port not open yet) → keep retrying
        # anything else = server is up but returned an error → fail immediately
        if [ -n "$http_code" ] && [ "$http_code" != "000" ]; then
            printf "\n❌ HTTP %s — server is up but returned an error:\n" "$http_code"
            cat /tmp/q2git_out
            printf "\n"
            exit 1
        fi
        printf "."; sleep 1
    done
    printf " ready!\n\n📊 Query execution (without commit)...\n%s\n" "$(cat /tmp/q2git_out)"
    printf "\n🚀 Testing commit...\n"
    curl -s -X POST "http://localhost:8000/api/commit"
    printf "\n✅ Test complete\n\n"

# Start wasmCloud host in detached mode (no-op if already running)
[group('environment')]
up:
    #!/bin/bash
    # Strip ANSI escape codes before grepping for an actual host ID (40+ uppercase
    # alphanumeric chars) — color codes break consecutive character matching
    if wash get hosts 2>/dev/null | sed $'s/\033\[[0-9;]*[mK]//g' | grep -qE '[A-Z0-9]{40,}'; then
        echo "wasmCloud host already running, skipping."
    else
        wash up -d
    fi

# Stop wasmCloud host
[group('environment')]
down:
    wash down

# Build the WebAssembly component
[group('app')]
build:
    wash build

# Push wasmCloud config (safe to commit — no secrets)
[group('app')]
config:
    #!/bin/bash
    config_value=$(cat <<'YAML'
    settings:
      write_mode: "append"

    source:
      method: "GET"
      headers:
        User-Agent: "q2git/1.0"
        Accept: "application/json"

    queries:
      - name: "power-consumption"
        description: "Daily power consumption difference (scalar query)"
        url: "http://prometheus-server.prometheus.svc.cluster.local:82/prometheus/api/v1/query?query=round%28%28sum%28sum_over_time%28sunwoda_pv1_power_w%5B22h%5D%29%29%2Bsum%28sum_over_time%28sunwoda_pv2_power_w%5B22h%5D%29%29%29*30%2F3600000%2C0.01%29"
        query: |
          if .status == "success" and .data.resultType == "vector" and (.data.result | length) >= 1 then
            "\n" + (.data.result[0].value[0] | tonumber | strftime("%Y-%m-%d %H:%M:%S")) + ", " + (if .data.result[0].value[1] == "NaN" then "No Data" else (.data.result[0].value[1] | tostring) end)
          else
            error("Invalid response format")
          end

    destination:
      api_url: "https://api.github.com"
      owner: "mihaigalos"
      repo: "test"
      branch: "main"
      output_path: "data_powerreadings.csv"
      commit_message: "Test from q2git"
    YAML
    )
    wash config put q2git-config "Q2GIT_CONFIG=${config_value}"

# Deploy the application
[group('app')]
deploy: build config
    wash app deploy wadm.yaml

# Push the built component to an OCI registry
[group('app')]
push registry=registry tag=tag: build
    #!/bin/bash
    set -euo pipefail
    helper=$(python3 -c "import json; d=json.load(open('$HOME/.docker/config.json')); print(d.get('credsStore',''))")
    creds=$(echo "https://index.docker.io/v1/" | docker-credential-"$helper" get)
    user=$(echo "$creds" | python3 -c "import json,sys; print(json.load(sys.stdin)['Username'])")
    pass=$(echo "$creds" | python3 -c "import json,sys; print(json.load(sys.stdin)['Secret'])")
    wash push --user "$user" --password "$pass" \
        {{registry}}/{{app}}:{{tag}} build/{{app}}_s.wasm

# Undeploy the application
[group('app')]
undeploy:
    wash app undeploy {{app}}

# List deployed applications
[group('app')]
list:
    wash app list

# View wasmCloud host logs
[group('observe')]
logs:
    tail -50 ~/.local/share/wash/downloads/wasmcloud.log

# Follow wasmCloud host logs
[group('observe')]
logs-follow:
    tail -f ~/.local/share/wash/downloads/wasmcloud.log

# Generate OpenAPI spec and serve it via Swagger UI
[group('observe')]
openapi:
    #!/bin/bash
    set -euo pipefail
    ROOT="$(pwd)"
    (cd scripts && . .venv/bin/activate && pip install -q -r requirements.txt && python3 generate_openapi.py)
    printf "🌐 Swagger UI → http://localhost:8081  (Ctrl+C to stop)\n"
    docker run --rm -p 8081:8080 \
        -e SWAGGER_JSON=/openapi.yaml \
        -v "$ROOT/openapi.yaml:/openapi.yaml:ro" \
        swaggerapi/swagger-ui

# Get detailed app status
[group('observe')]
status:
    wash app get {{app}}
