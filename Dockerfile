FROM --platform=$BUILDPLATFORM golang:1.15

ARG BUILDPLATFORM
ARG TARGETARCH
ARG TARGETOS
WORKDIR /go/src/github.com/wish/targetsync/
COPY . /go/src/github.com/wish/targetsync/
RUN CGO_ENABLED=0 GOARCH=${TARGETARCH} GOOS=${TARGETOS} go build -a -installsuffix cgo ./cmd/targetsync




FROM alpine:3.12
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/wish/targetsync/targetsync .
CMD /root/targetsync
