# Stage 1: Build
FROM golang:1.23-alpine AS build

RUN apk add --no-cache git

WORKDIR /build

# Copy the API library (needed for the replace directive in go.mod)
COPY lib/sq-api-go/ /build/lib/sq-api-go/

# Copy the migration tool source
COPY go/ /build/go/

WORKDIR /build/go
RUN go mod download
RUN CGO_ENABLED=0 go build -o /build/sonar-migration-tool .

# Stage 2: Runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=build /build/sonar-migration-tool /usr/local/bin/sonar-migration-tool

RUN mkdir -p /app/files && chown appuser:appgroup /app/files
WORKDIR /app

USER appuser

LABEL org.opencontainers.image.source=https://github.com/sonar-solutions/sonar-migration-tool

ENTRYPOINT ["sonar-migration-tool"]
