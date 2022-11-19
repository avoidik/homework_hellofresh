FROM golang:1.18-bullseye AS build

ARG CGO_ENABLED=1
ARG GOOS=linux
ARG GOARCH=amd64
ARG TAG_RELEASE=dev

RUN DEBIAN_FRONTEND=noninteractive apt-get -q -y update && \
    DEBIAN_FRONTEND=noninteractive apt-get -q -y install build-essential

WORKDIR /go/src/app
COPY *.go ./
COPY go.mod go.sum ./
RUN go mod download -x
RUN go test -v
RUN echo "Tagging build as $TAG_RELEASE" && \
    go build \
    -a \
    -tags "netgo" \
    -ldflags="-w -s -linkmode external -extldflags '-static' -X 'main.tagRelease=$TAG_RELEASE'" \
    -o /go/bin/fresh-server

FROM gcr.io/distroless/static-debian11 as final

COPY --from=build --chown=nonroot:nonroot /go/bin/fresh-server /app/fresh-server
WORKDIR /app
USER nonroot
EXPOSE 8080
CMD ["/app/fresh-server"]
