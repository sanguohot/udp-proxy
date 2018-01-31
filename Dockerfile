FROM scratch
WORKDIR /opt
ADD .tmp/* /opt/
ADD ca-certificates.crt /etc/ssl/certs/
EXPOSE 2022
ENTRYPOINT ["/opt/main"]