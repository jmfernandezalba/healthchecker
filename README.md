# SRE Fun coding test

## Execute with docker-compose

```
docker-compose up --build
```

## Execute with golang

1. Install dependencies

  ```
  go get ./...
  ```

2. Run program

  1. From source

  ```
  go run cmd/healthchecker/main.go test/testdata/healthCheckConfig.yaml
  ```

  2. From binary

  ```
  go build ./...
  ./healthchecker test/testdata/healthCheckConfig.yaml
  ```

## Execute tests and see coverage

```
go test -coverprofile cover.out ./...
go tool cover -html=cover.out
```

## Edit the config file to set Authorization header

In order to execute the notification to the notifier service, the authorization
header has to be edited to set the correct token.

```
notify:
  endpoint: "https://interview-notifier-svc.spotahome.net/api/v1/notification"
  method: POST
  header:
    authorization: Bearer CHANGEME
  body: "{\"service\": \"{{.FailingService}}\", \"description\": \"{{.FailingServiceDescription}}\"}"
```
