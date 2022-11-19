#!/bin/bash

#
# tag release based on branch and commit hash
#
TAG_RELEASE="$(git rev-parse --abbrev-ref HEAD)-$(git rev-parse --short=8 HEAD)"

#
# build docker container image
#
docker build -t fresh-server --build-arg TAG_RELEASE="$TAG_RELEASE" .

#
# optional parameter to run docker container
#
if [[ "$1" == "run" ]]; then
  docker run -i -t --rm --name fresh-server fresh-server
fi
