FROM alpine
ENV HELM_CACHE_HOME=/tmp
COPY bin/ /app/
WORKDIR /app
ENTRYPOINT ["/app/kubegems"]
