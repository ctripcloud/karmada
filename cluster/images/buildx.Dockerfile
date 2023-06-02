FROM hub.cloud.ctripcorp.com/karrier/alpine:3.12.7

ARG BINARY
ARG TARGETPLATFORM

COPY ${TARGETPLATFORM}/${BINARY} /bin/${BINARY}
