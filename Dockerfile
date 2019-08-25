FROM golang:latest

ADD . .

RUN ["go", "build", "-o", "telma-server"]

EXPOSE 5000

CMD ["./telma-server"]