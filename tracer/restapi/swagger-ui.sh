#!/usr/bin/env bash
base=$(dirname $(readlink -f "$0"))
exec docker run -it --rm \
    -v $base/:/usr/share/nginx/html/spec/ \
    -p 8080:8080 \
    swaggerapi/swagger-ui
