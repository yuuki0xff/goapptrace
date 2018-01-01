#!/usr/bin/env bash
base=$(dirname $(readlink -f "$0"))

{
    sleep 2
    xdg-open 'http://localhost:8080/?url=./spec/api.yaml'
} &

exec docker run -it --rm \
    -v $base/:/usr/share/nginx/html/spec/ \
    -p 8080:8080 \
    swaggerapi/swagger-ui
