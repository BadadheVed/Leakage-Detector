FROM golang:1.24-alpine AS builder


WORKDIR /app


RUN apk add --no-cache git


COPY go.mod go.sum ./
RUN go mod download


COPY . .
RUN go build -o main main.go
# Build Stage
FROM alpine:latest 

WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/main .
COPY inventory.json .
COPY .env .env
EXPOSE 8080
ENV GIN_MODE=release
ENV PORT=8080
CMD ["./main"]
