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
    wash config put q2git-config "Q2GIT_CONFIG=$(cat config.yaml)"

# Push wasmCloud secrets (reads from src/secrets.yaml — gitignored, never committed)
[group('app')]
secrets:
    #!/bin/bash
    token=$(cat src/secrets.yaml | yq '.destination.token' -r)
    username=$(cat src/secrets.yaml | yq '.source.username' -r)
    password=$(cat src/secrets.yaml | yq '.source.password' -r)
    wash config put q2git-secrets \
        "Q2GIT_GITHUB_TOKEN=${token}" \
        "Q2GIT_SOURCE_USERNAME=${username}" \
        "Q2GIT_SOURCE_PASSWORD=${password}"

# Deploy the application
[group('app')]
deploy: build config secrets
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
