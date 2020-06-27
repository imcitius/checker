FROM centos

LABEL "repository" = "https://github.com/imcitius/checker"
LABEL "homepage" = "https://github.com/imcitius/checker"
LABEL "maintainer" = "Ilya Rubinchik <cit@2cit.ru>"

COPY build/checker /bin/checker
COPY docs/examples/google.yaml /config.yaml

CMD ["/bin/checker", "--config", "/config.yaml", "check"]

ENTRYPOINT ["/bin/checker"]
