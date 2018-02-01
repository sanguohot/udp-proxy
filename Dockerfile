FROM scratch
COPY ./.tmp/main /
COPY ca-certificates.crt /etc/ssl/certs/
EXPOSE 2022/udp
ENTRYPOINT ["/main"]