#!/bin/bash
set -Eeuo pipefail

# From https://hub.docker.com/r/redocly/redoc

docker pull redocly/redoc

docker run -it --rm -p 80:80 \
    -v "$PWD"/api/:/usr/share/nginx/html/swagger/ \
    -e SPEC_URL=swagger/openapi.yaml \
    redocly/redoc
