FROM golang:1.23.3-alpine3.20 AS builder
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download
COPY . /app/
RUN go build -o tupyrae cmd/main.go

LABEL org.opencontainers.image.source=https://github.com/digode/tupyrae

FROM gcr.io/distroless/static@sha256:9be3fcc6abeaf985b5ecce59451acbcbb15e7be39472320c538d0d55a0834edc AS app
COPY --from=builder /app/tupyrae /bin/tupyrae

USER 65534

ENTRYPOINT ["/bin/tupyrae"]