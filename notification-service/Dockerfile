FROM golang:1.25-alpine AS builder

ENV GOTOOLCHAIN=local

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o service ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/service .
CMD ["./service"]