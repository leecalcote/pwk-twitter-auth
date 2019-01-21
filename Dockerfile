FROM golang:1.11.4 as bd
ADD . $GOPATH/src/github.com/leecalcote/pwk-twitter-auth/
WORKDIR $GOPATH/src/github.com/leecalcote/pwk-twitter-auth/
RUN go build -a -o /pwk-twitter-auth .

FROM alpine
RUN apk --update add ca-certificates
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY --from=bd /pwk-twitter-auth /app/
CMD /app/pwk-twitter-auth
