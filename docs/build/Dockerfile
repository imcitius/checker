FROM golang:1.17-alpine as build
COPY . /app
WORKDIR /app
RUN go get -v -d ./... \
    && CGO_ENABLED=0 GOOS=linux go build -v -i -ldflags "-X my/checker/config.Version=${SOURCE_VERSION}" -o build/checker

FROM alpine
RUN apk update && apk install curl
COPY --from=build /app/build/checker /bin/checker
COPY --from=build /app/docs/examples/google.yaml /config.yaml
ENTRYPOINT "/bin/checker"
