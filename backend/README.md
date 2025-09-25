# Stolos Platform Backend

## Setup

1. Install Go
2. Clone repository
3. Install dependencies:

    ```bash
    go mod download
    ```

## Configuration

Move .env.template to .env and adjust settings as needed.

## Run

If env `DB_HOST` is not set, it defaults to sqlite.

```bash
go run cmd/server/main.go
```

To use Postgres, set:

```bash
export DB_HOST=localhost
export DB_PASSWORD=postgres
go run cmd/server/main.go
```

## Build

```bash
go build -o out/server ./cmd/server
./out/server
```

## Docker

```bash
docker build -t stolos-platform-backend .
docker run -p 8080:8080 stolos-platform-backend
```

### Compose

```bash
docker-compose -f docker-compose.yml up
```

## API Testing

```bash
# Health Check
curl http://localhost:8080/health

# Initialize GCP (creates storage bucket and saves config):
curl -X POST http://localhost:8080/api/v1/gcp/initialize

# Check GCP status:
curl http://localhost:8080/api/v1/gcp/status

# List nodes:
curl http://localhost:8080/api/v1/nodes

# Create nodes:
curl -X POST http://localhost:8080/api/v1/nodes

# Get specific node:
curl http://localhost:8080/api/v1/nodes/uuid-here

# Sync GCP nodes (queries all GCP instances):
curl -X POST http://localhost:8080/api/v1/nodes/sync-gcp

# Generate ISO:
curl -X POST http://localhost:8080/api/v1/isos/generate
```

## Tests

```bash
go test ./tests/
```
