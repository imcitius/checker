FROM golang:1.17-alpine as build
COPY . /app
WORKDIR /app
RUN apk update && apk add gcc g++
RUN go get -v -d ./... \
    && CGO_ENABLED=1 GOOS=linux go build -ldflags "-X my/checker/config.Version=${GITHUB_REF_NAME}" -o build/checker

FROM alpine

LABEL "repository" = "https://github.com/imcitius/checker"
LABEL "homepage" = "https://github.com/imcitius/checker"
LABEL "maintainer" = "Ilya Rubinchik <citius@citius.dev>"

RUN apk update && apk add curl

COPY --from=build build/checker /bin/checker
COPY --from=build docs/examples/google.yaml /
COPY --from=build docs/build/entrypoint.sh /

ENTRYPOINT ["sh", "-c"]
CMD ["/entrypoint.sh"]
