FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git build-base
COPY backend/go.mod backend/go.sum ./backend/
COPY bootstrap/go.mod bootstrap/go.sum ./bootstrap/
RUN cd backend && go mod download
COPY backend/ ./backend/
COPY bootstrap/ ./bootstrap/
ARG TARGETOS TARGETARCH

RUN cd backend && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o main ./cmd/server

FROM --platform=$BUILDPLATFORM alpine:3.20 AS tf
ARG TF_VERSION=1.13.3
ARG TARGETOS TARGETARCH
RUN apk add --no-cache wget unzip ca-certificates openssl
WORKDIR /tmp
RUN set -eux; \
    TF_FILE="terraform_${TF_VERSION}_${TARGETOS}_${TARGETARCH}.zip"; \
    wget -q "https://releases.hashicorp.com/terraform/${TF_VERSION}/${TF_FILE}"; \
    wget -q "https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_SHA256SUMS"; \
    grep " ${TF_FILE}\$" "terraform_${TF_VERSION}_SHA256SUMS" | sha256sum -c -; \
    unzip -q "$TF_FILE" -d /out; \
    rm -f "$TF_FILE" "terraform_${TF_VERSION}_SHA256SUMS"

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libc-bin \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/backend/main .
COPY --from=tf /out/terraform /usr/local/bin/terraform
COPY backend/terraform-templates ./terraform-templates
EXPOSE 8080

CMD ["./main"]
