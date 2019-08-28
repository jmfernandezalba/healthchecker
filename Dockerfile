FROM golang:1.11-alpine3.9 AS build

RUN apk add --update \
    git \
 && rm /var/cache/apk/*

WORKDIR /go/src/github.com/jmfernandezalba/healthchecker

RUN go get gopkg.in/yaml.v2

COPY . .

RUN go build ./...

FROM alpine:3.9

COPY test/testdata /etc/healthchecker
COPY --from=build /go/src/github.com/jmfernandezalba/healthchecker/healthchecker /usr/local/bin

CMD ["healthchecker", "/etc/healthchecker/healthCheckConfig.yaml"]
