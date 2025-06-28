FROM alpine as certs
RUN apk update && apk add ca-certificates

FROM  --platform=linux/amd64 ubuntu

MAINTAINER Parth Mudgal <artpar@gmail.com>
WORKDIR /opt/daptin

COPY --from=certs /etc/ssl/certs /etc/ssl/certs

COPY daptin-linux-amd64 /opt/L3m0nSo/Memories
RUN chmod +x /opt/L3m0nSo/Memories
RUN ls -lah /opt/L3m0nSo/Memories



# Install glibc
#RUN apk add --force-overwrite --no-cache bash curl \
#    && curl -Lo /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub \
#    && curl -Lo /tmp/glibc.apk https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.35-r0/glibc-2.35-r0.apk \
#    && apk add --force-overwrite /tmp/glibc.apk

#RUN apk --force-overwrite add libc6-compat
#RUN apk add gcompat


EXPOSE 8080
ENTRYPOINT ["/opt/L3m0nSo/Memories", "-runtime", "release", "-port", ":8080"]