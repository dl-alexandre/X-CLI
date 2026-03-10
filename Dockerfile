FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY x /usr/local/bin/x
RUN chmod +x /usr/local/bin/x

RUN mkdir -p /root/.config/x-cli

ENV XDG_CONFIG_HOME=/root/.config
ENV HOME=/root

ENTRYPOINT ["x"]
CMD ["--help"]
