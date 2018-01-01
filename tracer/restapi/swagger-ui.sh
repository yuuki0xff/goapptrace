#!/usr/bin/env bash
base=$(dirname $(readlink -f "$0"))
exec docker run -it --rm \
    -v $base/:/srv/ \
    -e SWAGGER_JSON=/srv/api.yaml \
    -p 8080:8080 \
    swaggerapi/swagger-ui
