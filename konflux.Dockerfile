# Konflux hermetic build for the kubectl-oadp download server
# Dependencies are prefetched by the Konflux pipeline (cachi2) and injected
# into the build context before this Dockerfile runs.

FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25 AS builder

COPY . /workspace
WORKDIR /workspace

# Build release archives for all platforms (CGO_ENABLED=0 for cross-platform
# portability — CLI binaries run on user machines outside the FIPS boundary)
RUN make release-archives && \
    mkdir -p /archives && \
    mv *.tar.gz *.sha256 /archives/ && \
    rm -rf /root/.cache/go-build /tmp/* release-build/

# Build the download server (FIPS-compliant, runs in-cluster on RHEL)
RUN CGO_ENABLED=1 GOEXPERIMENT=strictfipsruntime GOOS=linux \
    go build -mod=readonly -a -tags strictfipsruntime \
    -o /workspace/bin/download-server ./cmd/downloads/ && \
    go clean -cache -modcache -testcache && \
    rm -rf /root/.cache/go-build /go/pkg

FROM registry.redhat.io/ubi9/ubi:latest

RUN dnf -y install openssl && dnf -y reinstall tzdata && dnf clean all

COPY --from=builder /archives /archives
COPY --from=builder /workspace/bin/download-server /usr/local/bin/download-server
COPY LICENSE /licenses/

EXPOSE 8080

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/download-server"]

LABEL description="OADP CLI - Binary Download Server"
LABEL io.k8s.description="OADP CLI - Binary Download Server"
LABEL io.k8s.display-name="OADP CLI Downloads"
LABEL io.openshift.tags="oadp,migration,backup"
LABEL summary="Serves pre-built kubectl-oadp binaries for all platforms"
