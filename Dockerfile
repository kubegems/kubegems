# https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
FROM --platform=${BUILDPLATFORM} golang:1 AS build
WORKDIR /src
COPY . .
# TARGETOS TARGETARCH already set by '--platform'
ARG TARGETOS TARGETARCH 
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    OS=${TARGETOS} ARCH=${TARGETARCH} make build

FROM alpine
COPY --from=build /src/bin /app
COPY deploy/plugins/ /app/plugins/
COPY deploy/*.yaml /app/plugins/
COPY config/promql_tpl.yaml /app/config/
COPY config/dashboards/ /app/config/dashboards/
WORKDIR /app
ENTRYPOINT ["/app/kubegems"]