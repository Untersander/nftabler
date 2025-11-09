FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum *.go ./
RUN go build -o /nftabler .

FROM alpine:3.20
RUN apk add --no-cache nftables
COPY --from=builder nftabler /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/nftabler"]