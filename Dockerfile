# Stage 1: Build frontend
FROM node:24-alpine AS frontend-build
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS go-build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY frontend_embed.go ./
COPY --from=frontend-build /build/frontend/dist frontend/dist/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /build/app ./cmd/api

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -g 1000 -S appgroup && adduser -u 1000 -S appuser -G appgroup
WORKDIR /app
RUN mkdir -p /app/data && chown appuser:appgroup /app/data
COPY --from=go-build /build/app /app/app
LABEL org.opencontainers.image.source="https://github.com/wbergg/advanced-systembolaget-system"
LABEL org.opencontainers.image.description="Advanced Systembolaget System"
USER appuser
EXPOSE 8080
CMD ["/app/app"]
