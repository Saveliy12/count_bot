# syntax=docker/dockerfile:1.7

FROM golang:1.26.9-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/expenses ./cmd

FROM alpine:3.22

ARG APP_UID=10001
ARG APP_GID=10001

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g "$APP_GID" app \
    && adduser -S -G app -u "$APP_UID" -h /app app

WORKDIR /app

COPY --chown=app:app --from=build /out/expenses /app/expenses

USER app

EXPOSE 8083

STOPSIGNAL SIGTERM

ENTRYPOINT ["/app/expenses"]
