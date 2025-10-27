# Testing

## Unit tests

You can run unit tests with the following command on the root of the project:
```sh
go test -cover -v ./...
```

## Integration tests

You can run integration tests with the following command on the root of the project:
```sh
go test -tags=integration -cover -v ./...
```
