FROM centos

LABEL "repository" = "https://github.com/imcitius/checker"
LABEL "homepage" = "https://github.com/imcitius/checker"
LABEL "maintainer" = "Ilya Rubinchik <citius@citius.dev>"

COPY checker-amd64 /bin/checker
COPY docs/examples/google.yaml /config.yaml

ENTRYPOINT ["/bin/checker", "check", "-c", "/config.yaml"]
