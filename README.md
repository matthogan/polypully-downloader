# downloader

Microservice for downloading resources. Receives URLs and returns status responses. 

Clients may poll for the status of the download until it has completed or listen for
events preferably.

Events are published to the configured event bus when downloads are started and completed or when an
error occurs.

The service stores the downloaded resources in the configured storage backend.

The service is long running and services a single request at a time unless configured otherwise.

It will not not resume downloads after a restart.

Exposes a simple REST API defined as an OpenAPI specification.

## API

[OpenAPI Spec](/api/openapi.yaml)

## Building

<https://goreleaser.com/>

```shell
brew install goreleaser make
```

Linux build

```shell
goreleaser build --snapshot --rm-dist --id linux
```

or

```shell
make linux-build
```

## Release

Releasing requires an api token. Github tokens can be generated
in [Developer settings](https://github.com/settings/tokens).

```shell
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
GITHUB_TOKEN="..." goreleaser release
```

or

```shell
make release TAG=0.1.0
```

# Re/Generate API

Requires: npm, openapi-generator-cli, java

```shell

```shell
openapi-generator-cli generate -i api/openapi/openapi.yaml -g go-server -o api/openapi/generated
```

