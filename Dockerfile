FROM golang:1.12-alpine3.10 AS build

RUN apk add --update \
    git \
 && rm /var/cache/apk/*

WORKDIR /go/src/github.com/jmfernandezalba/healthchecker

RUN go get gopkg.in/yaml.v2

COPY . .

RUN go build ./...

FROM alpine:3.10

RUN apk add --update \
    ca-certificates \
 && rm /var/cache/apk/*

COPY test/testdata /etc/healthchecker
COPY --from=build /go/src/github.com/jmfernandezalba/healthchecker/healthchecker /usr/local/bin

CMD ["healthchecker", "/etc/healthchecker/healthCheckConfig.yaml"]
