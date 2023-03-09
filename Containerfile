FROM registry.access.redhat.com/ubi8/go-toolset:1.18

USER root

WORKDIR /test

# Install dependencies: `operator-sdk`
ARG OPERATOR_SDK_VERSION=v1.26.0
ARG OPERATOR_SDK_URL=https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}
RUN cd /usr/local/bin \
    && curl -LO ${OPERATOR_SDK_URL}/operator-sdk_linux_amd64 \
    && mv operator-sdk_linux_amd64 operator-sdk \
    && chmod +x operator-sdk

# Install dependencies: `golangci-lint & ginkgo`
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1
RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.9.0

RUN mkdir /test-run-results && chmod 777 /test-run-results

ENV ARTIFACT_DIR=/test-run-results
ENV GOLANGCI_LINT_CACHE=/tmp/.cache
ENV GOCACHE=/tmp/
ENV PATH="${PATH}:/opt/app-root/src/go/bin"

COPY ./ .

RUN chmod 777 /test -R

#ENTRYPOINT ["make"]
