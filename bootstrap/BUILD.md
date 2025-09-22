# Build instruction

```bash
CGO_ENABLED=0 go build -ldflags='-extldflags="-static" -X github.com/stolos-cloud/stolos-bootstrap/pkg/github.GithubClientId=xxxx -X github.com/stolos-cloud/stolos-bootstrap/pkg/github.GithubClientSecret=xxxx -X github.com/stolos-cloud/stolos-bootstrap/pkg/gcp.GCPClientId=xxxx -X github.com/stolos-cloud/stolos-bootstrap/pkg/gcp.GCPClientSecret=xxxx' -o out/bootstrap ./cmd/bootstrap
```
