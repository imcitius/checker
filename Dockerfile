FROM debian

LABEL "repository" = "https://github.com/imcitius/checker"
LABEL "homepage" = "https://github.com/imcitius/checker"
LABEL "maintainer" = "Ilya Rubinchik <citius@citius.dev>"

COPY checker /bin/checker
COPY scripts/fly-deploy/entrypoint.sh /
COPY .helm/envs/fly /

ENTRYPOINT ["sh", "-c"]
CMD ["/entrypoint.sh"]
