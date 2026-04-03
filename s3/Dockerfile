FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

# Download dependencies
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o ./main ./cmd/server/main.go

# Use a minimal base image for the final container
FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]