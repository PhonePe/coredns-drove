FROM golang:1.20 as build
ENV SSL_CERT_DIR=/etc/ssl/certs
RUN apt-get update && apt-get install -y ca-certificates openssl


RUN openssl s_client -showcerts -connect github.com:443  </dev/null 2>/dev/null|openssl x509 -outform PEM > ${SSL_CERT_DIR}/github.crt
RUN openssl s_client -showcerts -connect proxy.golang.org:443 </dev/null 2>/dev/null|openssl x509 -outform PEM > ${SSL_CERT_DIR}/proxy.golang.crt
RUN openssl s_client -showcerts -connect sum.golang.org:443 </dev/null 2>/dev/null|openssl x509 -outform PEM > ${SSL_CERT_DIR}/sum.golang.crt
# Update certificates
RUN update-ca-certificates

COPY . /opt/coredns-drove/
WORKDIR /opt/coredns-drove
RUN go build
FROM golang:1.20
COPY --from=build /opt/coredns-drove/coredns-drove /opt
COPY Corefile /opt/Corefile
WORKDIR /opt
EXPOSE 1053 1053/udp
ENTRYPOINT ["/opt/coredns-drove"]