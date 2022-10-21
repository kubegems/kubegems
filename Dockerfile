FROM --platform=${BUILDPLATFORM} alpine
# TARGETOS TARGETARCH already set by '--platform'
COPY bin /app
ARG TARGETOS TARGETARCH 
RUN ln -sf /app/kubegems-${TARGETOS}-${TARGETARCH} /app/kubegems
WORKDIR /app
ENTRYPOINT ["/app/kubegems"]