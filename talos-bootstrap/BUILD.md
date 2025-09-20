## Build instruction

```bash
CGO_ENABLED=0 go build -ldflags='-extldflags="-static" -X main.GithubClientId=xxxx -X main.GithubClientSecret=xxxx"' -o out/talos-bootstrap -a  main
```