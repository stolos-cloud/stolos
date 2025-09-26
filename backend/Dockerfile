FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev
COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -o main ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates git wget unzip

# Install Terraform
RUN wget https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_linux_amd64.zip && \
    unzip terraform_1.5.7_linux_amd64.zip && \
    mv terraform /usr/local/bin/ && \
    rm terraform_1.5.7_linux_amd64.zip

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/terraform-templates ./terraform-templates

EXPOSE 8080

CMD ["./main"]