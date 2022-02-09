# Build the manager binary
FROM golang:1 as builder
WORKDIR /workspace
COPY . .
RUN make build
# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/bin/kubegems .
USER nonroot:nonroot
ENTRYPOINT ["/kubegems"]
