#!/bin/sh
# Example:  ./hack/go-genmock.sh

if [ "$IS_CONTAINER" != "" ]; then
  go install go.uber.org/mock/mockgen
  go generate ./pkg/asset/installconfig/... "${@}"
  go generate ./pkg/destroy/... "${@}"
else
  podman run --rm \
    --env IS_CONTAINER=TRUE \
    --volume "${PWD}:/go/src/github.com/openshift/installer:z" \
    --workdir /go/src/github.com/openshift/installer \
    docker.io/golang:1.24 \
    ./hack/go-genmock.sh "${@}"
fi
