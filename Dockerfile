FROM alpine:3.9.4

WORKDIR /app

RUN mkdir -p /app/config

RUN adduser -S -D -H -h /app appuser

USER appuser

COPY ./main /app/

COPY ./config/config.json /app/config/

EXPOSE 8080

ENTRYPOINT ["./main"]
