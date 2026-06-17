# Stage 1: Build frontend
FROM oven/bun:latest AS frontend
WORKDIR /src/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

# Stage 2: Build Go binary
FROM golang:1.25 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./internal/server/static
RUN CGO_ENABLED=0 go build -o /xpo ./cmd/exponential

# Stage 3: Minimal runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates git
COPY --from=builder /xpo /usr/local/bin/xpo
WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["xpo", "serve"]
