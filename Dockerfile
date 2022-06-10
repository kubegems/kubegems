FROM alpine
COPY bin/ /app/
COPY deploy/plugins/ /app/plugins/
COPY deploy/*.yaml /app/plugins/
WORKDIR /app
ENTRYPOINT ["/app/kubegems"]
