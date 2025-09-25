FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev
COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -o main ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates git

WORKDIR /root/

COPY --from=builder /app/main .

RUN mkdir -p /root/configs /root/terraform-workspace /root/gitops-workspace

EXPOSE 8080

CMD ["./main"]