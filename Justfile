# Justfile for wasmCloud hello-world application

# Set PATH to use the native-tls version of wash
export PATH := env_var('HOME') + "/.cargo/bin:" + env_var('PATH')

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
    printf "\n⏳ Waiting for component to be ready (retrying until outgoing HTTP is live)...\n"
    until result=$(curl -sf -X POST http://localhost:8000/api/execute 2>/dev/null); do printf "."; sleep 1; done
    printf " ready!\n\n📊 Query execution (without commit)...\n%s\n" "$result"
    printf "\n🚀 Testing commit...\n"
    curl -s -X POST "http://localhost:8000/api/commit"
    printf "\n✅ Test complete\n\n"

# Start wasmCloud host in detached mode (no-op if already running)
[group('environment')]
up:
    #!/bin/bash
    if wash get hosts 2>/dev/null | grep -q .; then
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

# Deploy the application
[group('app')]
deploy: build
    wash app deploy wadm.yaml

# Undeploy the application
[group('app')]
undeploy:
    wash app undeploy q2git

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
    wash app get q2git
