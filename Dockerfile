FROM alpine
COPY deploy/charts /app/charts
COPY deploy/plugins /app/plugins
ENV HELM_CACHE_HOME=/tmp
COPY bin/kubegems /app/kubegems
ENTRYPOINT ["/app/kubegems"]
