FROM alpine
WORKDIR /serv
COPY bin/kubegems /serv
ENTRYPOINT ["/serv/kubegems"]
