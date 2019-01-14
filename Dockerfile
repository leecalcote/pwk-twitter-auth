FROM golang:1.11.4 as bd
ADD . $GOPATH/src/github.com/leecalcote/pwk-twitter-auth/
WORKDIR $GOPATH/src/github.com/leecalcote/
RUN go build -a -o /pwk-twitter-auth .

FROM ubuntu
RUN apt-get update; apt-get install -y ca-certificates; update-ca-certificates
COPY --from=bd /pwk-twitter-auth /app/
CMD /app/pwk-twitter-auth
