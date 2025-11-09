FROM alpine:3.20
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache nftables

COPY linux/${TARGETARCH}/nftabler /usr/local/bin/nftabler

RUN chmod +x /usr/local/bin/nftabler
ENTRYPOINT ["/usr/local/bin/nftabler"]