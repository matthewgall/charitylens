# Multi-stage build for CharityLens
# Stage 1: Build the application
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the main application with static linking where possible
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -ldflags '-extldflags "-static"' \
    -tags 'netgo osusergo sqlite_omit_load_extension' \
    -o charitylens ./cmd/charitylens

# Build the seeder tool (optional, for data import)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -ldflags '-extldflags "-static"' \
    -tags 'netgo osusergo sqlite_omit_load_extension' \
    -o charityseeder ./cmd/charityseeder

# Build a minimal healthcheck binary
RUN printf 'package main\nimport("net/http";"os")\nfunc main(){r,_:=http.Get("http://localhost:"+os.Getenv("PORT")+"/");if r!=nil&&r.StatusCode==200{os.Exit(0)};os.Exit(1)}' > healthcheck.go && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o healthcheck healthcheck.go

# Stage 2: Create minimal runtime image using scratch-like alpine
FROM alpine:latest

# Install ONLY the absolute minimum runtime dependencies
# ca-certificates: for HTTPS API calls
# tzdata: for timezone support
RUN apk add --no-cache ca-certificates tzdata && \
    rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 charitylens && \
    adduser -D -u 1000 -G charitylens charitylens

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/charitylens /app/charityseeder /app/healthcheck ./

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Copy web templates
COPY --from=builder /app/web ./web

# Create data directory for database
RUN mkdir -p /data && chown -R charitylens:charitylens /data /app

# Switch to non-root user
USER charitylens

# Expose port
EXPOSE 8080

# Environment variables with defaults
ENV DATABASE_TYPE=sqlite \
    DATABASE_URL=/data/charitylens.db \
    PORT=8080 \
    IP=0.0.0.0 \
    DEBUG=false

# Health check using our minimal healthcheck binary
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/healthcheck"]

# Run the application
CMD ["./charitylens"]
