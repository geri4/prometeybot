FROM instrumentisto/glide
WORKDIR /go/src/github.com/geri4/prometeybot/
COPY . .
RUN /usr/local/bin/glide install

FROM golang:1.10
WORKDIR /go/src/github.com/geri4/prometeybot/
COPY --from=0 /go/src/github.com/geri4/prometeybot/ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o prometeybot .

FROM alpine:3.5
LABEL maintainer "Andrey Gerasimov"
RUN apk add --no-cache ca-certificates
COPY --from=1 /go/src/github.com/geri4/prometeybot/prometeybot /prometeybot
RUN mkdir /data
VOLUME /data
EXPOSE 9010
ENTRYPOINT ["/prometeybot"]
