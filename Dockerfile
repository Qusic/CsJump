FROM alpine:edge
RUN apk --no-cache upgrade \
 && apk --no-cache add github-cli socat yq
COPY bin /usr/local/bin
WORKDIR /data
ENTRYPOINT [ "csjump" ]
