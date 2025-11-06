# Stolos Portal Backend

Backend service for the Stolos Cloud Portal.

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
go run cmd/server
```

## Build

```bash
go build -o out/server ./cmd/server
./out/server
```

## Docker

```bash
# Needs to build from root of the repository
cd ../
docker build -t stolos-platform-backend ../ -f Dockerfile
docker run -p 8080:8080 stolos-platform-backend
```

### Compose

```bash
docker-compose -f docker-compose.yml up
```

## API Documentation

Swagger UI available at: <http://localhost:8080/swagger/index.html>

To regenerate the Swagger docs after making changes to API annotations:

```bash
swag init -g cmd/server/main.go -o docs
```

## Tests

```bash
 go test ./...
```

## Populating GCP Resources

Run this script to fetch and cache available zones and machine types for the
configured region. This data is used to populate dropdowns in the VM
provisioning forms.

```bash
go run cmd/gcp-resources/main.go
```
