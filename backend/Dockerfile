FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /quietsignal ./cmd/server

FROM alpine:3.20
RUN adduser -D app && mkdir -p /data && chown app:app /data
USER app
COPY --from=builder /quietsignal /quietsignal
EXPOSE 8080
ENTRYPOINT ["/quietsignal"]
