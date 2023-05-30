FROM hub.cloud.ctripcorp.com/karrier/alpine:3.12.7

ARG BINARY
ARG TARGETPLATFORM

RUN mkdir -p /usr/local/bin

COPY ${TARGETPLATFORM}/${BINARY} /usr/local/bin/${BINARY}

ENTRYPOINT [ "/usr/local/bin/${BINARY}" ]
