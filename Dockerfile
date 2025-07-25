FROM alpine:latest

RUN apk --no-cache add ca-certificates docker-cli docker-compose

WORKDIR /root/

COPY dc-update .

ENTRYPOINT ["./dc-update"]