# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-trimpath \
	-ldflags="-s -w" \
	-o /out/udharo-pro-api ./cmd/api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget \
	&& addgroup -S app \
	&& adduser -S -D -H -h /app -s /sbin/nologin -G app app

WORKDIR /app

COPY --from=build /out/udharo-pro-api /app/udharo-pro-api

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
	CMD wget -qO- http://127.0.0.1:${APP_PORT:-8080}/health >/dev/null || exit 1

ENTRYPOINT ["/app/udharo-pro-api"]
