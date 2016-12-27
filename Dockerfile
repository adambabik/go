FROM golang:1.7-alpine

RUN apk add --no-cache git bash make

RUN mkdir -p /go/src/github.com/adambabik/go-collections
COPY . /go/src/github.com/adambabik/go-collections

COPY ./docker-entrypoint.sh /
RUN chmod 755 /docker-entrypoint.sh

WORKDIR /go/src/github.com/adambabik/go-collections

RUN make install && rm -rf /go/.cache
RUN go-wrapper install

EXPOSE 8888
ENTRYPOINT ["/docker-entrypoint.sh"]

CMD ["go-collections"]
