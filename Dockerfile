FROM alpine
COPY bin/ /app/
COPY deploy/plugins/ /app/plugins/
COPY deploy/*.yaml /app/plugins/
COPY config/promql_tpl.yaml /app/config/
COPY config/dashboards/ /app/config/dashboards/
WORKDIR /app
ENTRYPOINT ["/app/kubegems"]
