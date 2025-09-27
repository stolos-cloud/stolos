FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH

RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

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

FROM alpine:3.20
RUN adduser -D -H -u 10001 app && apk add --no-cache ca-certificates gcompat
WORKDIR /app
COPY --from=builder /out/server /app/server
COPY --from=tf /out/terraform /usr/local/bin/terraform

COPY terraform-templates ./terraform-templates
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
