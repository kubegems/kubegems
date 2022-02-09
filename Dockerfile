FROM alpine
WORKDIR /
COPY bin/kubegems .
ENTRYPOINT ["/kubegems"]
