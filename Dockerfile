# syntax=docker/dockerfile:1
FROM alpine
# TARGETOS TARGETARCH already set by '--platform'
ARG TARGETOS TARGETARCH 
COPY kubegems-${TARGETOS}-${TARGETARCH} /app/kubegems
COPY config /app/config
COPY plugins /app/plugins
COPY tools /app/tools
WORKDIR /app
ENTRYPOINT ["/app/kubegems"]
