# Justfile for wasmCloud hello-world application

# Set PATH to use the native-tls version of wash
export PATH := env_var('HOME') + "/.cargo/bin:" + env_var('PATH')

@_default:
    just --list
# Build the WebAssembly component
build:
    wash build

# Start wasmCloud host in detached mode
up:
    wash up -d

# Stop wasmCloud host
down:
    wash down

# Deploy the application
deploy: build
    wash app deploy wadm.yaml

# Undeploy the application
undeploy:
    wash app undeploy q2git

# Redeploy (undeploy + deploy)
redeploy: undeploy deploy

# List deployed applications
list:
    wash app list

# Test the deployed application
@test: redeploy
    echo "\n📊 Testing query execution (without commit)..."
    curl -s -X POST http://localhost:8000/api/execute | jq '{timestamp, result_count: (.results | length)}'
    echo "\n🚀 Testing query execution with git commit..."
    curl -s -X POST "http://localhost:8000/api/execute?commit=true" | jq '.'
    echo "\n✅ Test complete\n"

# Get detailed app status
status:
    wash app get q2git

# View wasmCloud host logs
logs:
    tail -50 ~/.local/share/wash/downloads/wasmcloud.log

# Follow wasmCloud host logs
logs-follow:
    tail -f ~/.local/share/wash/downloads/wasmcloud.log

# Generate OpenAPI specification
generate-openapi:
    #!/bin/bash
    cd scripts && . .venv/bin/activate
    pip install -r requirements.txt
    python3 generate_openapi.py

# Full development cycle: build, deploy, and test
dev: up deploy
    @echo "✅ Application deployed! Testing..."
    @sleep 3
    @just test

# Setup wasm32-wasip2 target for TinyGo (one-time setup for macOS ARM64)
_setup_wasm:
    #!/bin/bash
    set -e
    TINYGO_VERSION=$(tinygo version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    TARGET_FILE="/opt/homebrew/Cellar/tinygo/${TINYGO_VERSION}/targets/wasm32-wasip2.json"
    if [ -f "$TARGET_FILE" ]; then echo "✅ wasm32-wasip2 target already exists"; exit 0; fi
    echo "📝 Creating wasm32-wasip2 target file at $TARGET_FILE"
    cat > "$TARGET_FILE" << 'EOF'
    {
        "llvm-target":   "wasm32-wasi",
        "cpu":           "generic",
        "features":      "+bulk-memory,+mutable-globals,+nontrapping-fptoint,+sign-ext,+reference-types",
        "build-tags":    ["tinygo.wasm", "wasi"],
        "goos":          "wasip1",
        "goarch":        "wasm",
        "linker":        "wasm-ld",
        "libc":          "wasi-libc",
        "rtlib":         "compiler-rt",
        "gc":            "conservative",
        "scheduler":     "asyncify",
        "default-stack-size": 16384,
        "cflags": [
            "-mbulk-memory",
            "-mnontrapping-fptoint",
            "-msign-ext",
            "-mreference-types"
        ],
        "ldflags": [
            "--allow-undefined",
            "--stack-first",
            "--no-demangle"
        ],
        "extra-files": [
            "src/runtime/asm_tinygowasm.S"
        ]
    }
    EOF
    echo "✅ wasm32-wasip2 target created successfully"
