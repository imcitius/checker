FROM centos

LABEL "repository" = "https://github.com/imcitius/checker"
LABEL "homepage" = "https://github.com/imcitius/checker"
LABEL "maintainer" = "Ilya Rubinchik <cit@2cit.ru>"

COPY build/checker /bin/checker

ENTRYPOINT ["/bin/checker"]
