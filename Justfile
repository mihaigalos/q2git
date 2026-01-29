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
test:
    @echo "\nTesting root endpoint..." && curl http://localhost:8000
    @echo "\nTesting health endpoint..." && curl http://localhost:8000/health
    @echo "\nTesting greet endpoint..." && curl 'http://localhost:8000/api/greet?name=Alice'
    @echo "\nTesting echo endpoint..." && curl http://localhost:8000/api/echo
    @echo "\nTesting time endpoint..." && curl http://localhost:8000/api/time
    @echo "\nTesting status endpoint..." && curl http://localhost:8000/api/status
    @echo "\nTesting 404 response..." && curl http://localhost:8000/unknown
    @echo "\n"

# Get detailed app status
status:
    wash app get q2git

# View wasmCloud host logs
logs:
    tail -50 ~/.local/share/wash/downloads/wasmcloud.log

# Follow wasmCloud host logs
logs-follow:
    tail -f ~/.local/share/wash/downloads/wasmcloud.log

# Clean build artifacts
clean:
    cargo clean
    rm -rf build/

# Full development cycle: build, deploy, and test
dev: up deploy
    @echo "✅ Application deployed! Testing..."
    @sleep 3
    @just test
