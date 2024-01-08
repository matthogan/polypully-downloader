# syntax=docker/dockerfile:1
ARG GO_VERSION=1.21.0
ARG OS_VERSION=3.15

# if git4lab8 did releases with artefacts this build would not be necessary here
FROM golang:${GO_VERSION}-alpine as build
RUN    apk update \
    && apk --no-cache add ca-certificates \
    && apk add git
RUN go install github.com/goreleaser/goreleaser@latest
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY pkg ./pkg
COPY .goreleaser.yml ./
# snapshot because gitlab needs an upgrade before releases will work
RUN goreleaser build --id linux --snapshot --rm-dist

FROM alpine:${OS_VERSION}
RUN apk update && apk upgrade
WORKDIR /app
COPY --from=build /app/dist/downloader-linux-amd64 /usr/bin/downloader
ENTRYPOINT [ "downloader" ]
