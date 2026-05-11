FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o release-notes-gen ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/release-notes-gen .
COPY web/ ./web/
COPY templates/ ./templates/
EXPOSE 8080
CMD ["./release-notes-gen"]
