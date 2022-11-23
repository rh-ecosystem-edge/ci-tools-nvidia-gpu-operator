FROM registry.access.redhat.com/ubi8/go-toolset:1.18.4

USER root

WORKDIR /test

RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45.2 
RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.5.1
RUN mkdir /test-run-results && chmod 777 /test-run-results

ENV ARTIFACT_DIR=/test-run-results
ENV GOLANGCI_LINT_CACHE=/tmp/.cache
ENV GOCACHE=/tmp/
ENV PATH="${PATH}:/opt/app-root/src/go/bin"

COPY ./ .

RUN chmod 777 /test -R

#ENTRYPOINT ["make"]
