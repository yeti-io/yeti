# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

ARG SERVICE_NAME
ARG VERSION=dev

LABEL maintainer="Yeti Project"
LABEL service="${SERVICE_NAME}"
LABEL version="${VERSION}"

WORKDIR /build

RUN apk update && apk add --no-cache \
    ca-certificates \
    git \
    make

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -a \
    -installsuffix cgo \
    -trimpath \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o /build/bin/runner \
    ./cmd/${SERVICE_NAME}

FROM alpine:3.21

ARG SERVICE_NAME
ARG UID=1000
ARG GID=1000

RUN apk update && apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

WORKDIR /app

RUN addgroup -g "${GID}" -S runner && \
    adduser -S runner -u "${UID}" -G runner && \
    mkdir -p /app && \
    chown -R runner:runner /app

COPY --from=builder --chown=runner:runner /build/bin/runner /app/runner

USER runner:runner

EXPOSE 8080

CMD ["/app/runner"]
