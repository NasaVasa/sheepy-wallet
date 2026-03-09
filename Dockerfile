FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o sheepy-wallet ./cmd/server/

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/sheepy-wallet .
COPY config.example.json ./config.json
EXPOSE 8000
CMD ["./sheepy-wallet"]
