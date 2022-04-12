FROM alpine
ENV HELM_CACHE_HOME=/tmp
COPY bin/ /app/
COPY deploy/plugins /app/plugins
WORKDIR /app
ENTRYPOINT ["/app/kubegems"]
