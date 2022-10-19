# https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
FROM --platform=${BUILDPLATFORM} golang:1 AS build
WORKDIR /src

# Workaround on: https://github.com/moby/buildkit/issues/1673
# The `--mount=type=cache` caches persistent in buildx container via mounts at path `/var/lib/buildkit`.
# It dose'not work on github actions,because of the buildx context always recreate and caches'll lost.
# As an alt, we cache deps in a 'cache layer', buildx'll reuse it.
COPY go.mod go.sum /src/
RUN go mod download

COPY . .
# TARGETOS TARGETARCH already set by '--platform'
ARG TARGETOS TARGETARCH 
RUN \
    # Comment on github actions
    # --mount=type=cache,target=/root/.cache/go-build \
    # --mount=type=cache,target=/go/pkg \
    OS=${TARGETOS} ARCH=${TARGETARCH} make build

FROM alpine
COPY --from=build /src/bin /app
WORKDIR /app
ENTRYPOINT ["/app/kubegems"]