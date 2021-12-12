FROM golang:alpine AS build
WORKDIR /app
COPY . .
RUN go build -o out/csjump

FROM alpine:edge
RUN apk --no-cache upgrade \
 && apk --no-cache add github-cli openssh-client
COPY --from=build /app/out /usr/local/bin
WORKDIR /data
ENTRYPOINT [ "csjump" ]
