#FROM golang:1.17 as build
#COPY . /app
#WORKDIR /app
##RUN go get -v -d ./... \
##    && CGO_ENABLED=1 GOOS=linux go build -ldflags "-X my/checker/config.Version=${GITHUB_REF_NAME}" -o build/checker

FROM debian

LABEL "repository" = "https://github.com/imcitius/checker"
LABEL "homepage" = "https://github.com/imcitius/checker"
LABEL "maintainer" = "Ilya Rubinchik <citius@citius.dev>"

RUN apt update && apt install -y curl && apt upgrade ca-certificates -y && apt-get clean

COPY ../../build/checker /bin/checker
COPY docs/examples/google.yaml /
COPY docs/build/entrypoint.sh /

ENTRYPOINT ["sh", "-c"]
CMD ["/entrypoint.sh"]
