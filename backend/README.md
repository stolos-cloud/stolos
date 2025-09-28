# Stolos Portal Backend

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

# Configure gcp service-account
source .env
curl -X PUT http://localhost:8080/api/v1/gcp/service-account \
    -H "Content-Type: application/json" \
    -d "$(jq -n \
      --arg project_id "$GCP_PROJECT_ID" \
      --arg region "$GCP_REGION" \
      --arg service_account_json "$GCP_SERVICE_ACCOUNT_JSON" \
      '{
        project_id: $project_id,
        region: $region, 
        service_account_json: $service_account_json
      }')"

# Create terraform bucket:
curl -X POST http://localhost:8080/api/v1/gcp/bucket -d '{
  "project_id": "your-project-id",
  "region": "your-region"
}'

# init infra
curl -X POST http://localhost:8080/api/v1/gcp/init-infra

# destroy infra
curl -X POST http://localhost:8080/api/v1/gcp/destroy-infra

# List nodes:
curl http://localhost:8080/api/v1/nodes

# List pending nodes:
curl http://localhost:8080/api/v1/nodes?status=pending

# Create nodes:
curl -X POST http://localhost:8080/api/v1/nodes

# Get specific node:
curl http://localhost:8080/api/v1/nodes/uuid-here

# Sync GCP nodes:
curl -X POST http://localhost:8080/api/v1/nodes/sync-gcp

# Generate ISO:
curl -X POST http://localhost:8080/api/v1/isos/generate


# Login:
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'

# Register new user:
curl -X POST http://localhost:8080/api/v1/auth/admin/register \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"userpass123"}'

# Get user profile:
curl -X GET http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Refresh token:
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Create team:
curl -X POST http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"developers"}'

# List teams:
curl -X GET http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Get team details:
curl -X GET http://localhost:8080/api/v1/teams/TEAM_UUID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Add user to team:
curl -X POST http://localhost:8080/api/v1/teams/TEAM_UUID/users \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"USER_UUID"}'

# Remove user from team:
curl -X DELETE http://localhost:8080/api/v1/teams/TEAM_UUID/users/USER_UUID \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"

# Create user as admin:
curl -X POST http://localhost:8080/api/v1/auth/admin/users \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"newuser@example.com","password":"password123","role":"developer"}'

# List all users:
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

## Tests

```bash
go test ./tests/
```
