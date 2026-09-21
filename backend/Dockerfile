# GonsGarage API — template based on Arnela backend Dockerfile.
FROM golang:1.27-alpine AS builder

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gonsgarage-api ./cmd/api

FROM alpine:3.21

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /app/gonsgarage-api ./gonsgarage-api
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./gonsgarage-api"]
