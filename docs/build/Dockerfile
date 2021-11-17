FROM golang:1.17 as build
COPY . /app
WORKDIR /app
RUN go get -v -d ./... \
    && CGO_ENABLED=1 GOOS=linux go build -v -i -ldflags "-X my/checker/config.Version=${SOURCE_VERSION}" -o build/checker

FROM debian

LABEL "repository" = "https://github.com/imcitius/checker"
LABEL "homepage" = "https://github.com/imcitius/checker"
LABEL "maintainer" = "Ilya Rubinchik <citius@citius.dev>"

RUN apt update && apt install -y curl && apt upgrade ca-certificates -y && apt-get clean

COPY --from=build /app/build/checker /bin/checker
COPY --from=build /app/docs/examples/google.yaml /

COPY docs/build/entrypoint.sh /
COPY docs/examples/fly-ams.yaml /
ENTRYPOINT ["sh", "-c"]
CMD ["/entrypoint.sh"]
