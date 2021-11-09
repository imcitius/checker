FROM alpine

LABEL "repository" = "https://github.com/imcitius/checker"
LABEL "homepage" = "https://github.com/imcitius/checker"
LABEL "maintainer" = "Ilya Rubinchik <citius@citius.dev>"

COPY /builds/sysadmin/checker/build/checker /bin/checker
COPY scripts/fly-deploy/entrypoint.sh /
COPY scripts/fly-deploy/config /

ENTRYPOINT ["sh", "-c"]
CMD ["/entrypoint.sh"]
