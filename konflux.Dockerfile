# Konflux hermetic build for the kubectl-oadp download server
# Dependencies are prefetched by the Konflux pipeline (cachi2) and injected
# into the build context before this Dockerfile runs.

FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25 AS builder

COPY . /workspace
WORKDIR /workspace

# Version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown

# Build release binaries for all platforms (CGO_ENABLED=0 for cross-platform
# portability — CLI binaries run on user machines outside the FIPS boundary)
RUN set -e && \
    mkdir -p /archives && \
    for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do \
        os=$(echo $platform | cut -d'/' -f1) && \
        arch=$(echo $platform | cut -d'/' -f2) && \
        if [ "$os" = "windows" ]; then \
            output="kubectl-oadp_${os}_${arch}.exe"; \
        else \
            output="kubectl-oadp_${os}_${arch}"; \
        fi && \
        echo "Building $output..." && \
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch \
            go build -trimpath -mod=mod \
            -ldflags="-s -w \
                -X github.com/vmware-tanzu/velero/pkg/buildinfo.Version=${VERSION} \
                -X github.com/vmware-tanzu/velero/pkg/buildinfo.GitSHA=${GIT_COMMIT} \
                -X github.com/vmware-tanzu/velero/pkg/buildinfo.GitTreeState=clean" \
            -o /archives/$output \
            . && \
        (cd /archives && sha256sum $output > $output.sha256); \
    done && \
    cp LICENSE /archives/LICENSE && \
    rm -rf /root/.cache/go-build /tmp/*

# Build the download server (FIPS-compliant, runs in-cluster on RHEL)
RUN CGO_ENABLED=1 GOEXPERIMENT=strictfipsruntime GOOS=linux \
    go build -trimpath -mod=mod -a -tags strictfipsruntime \
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
