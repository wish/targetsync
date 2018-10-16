FROM golang:1.11
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR /go/src/github.com/wish/targetsync/
COPY . /go/src/github.com/wish/targetsync/
RUN dep ensure
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo ./cmd/targetsync




FROM alpine:3.7
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/wish/targetsync/targetsync .
CMD /root/targetsync
