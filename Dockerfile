FROM alpine:latest

# Build-time metadata as defined at http://label-schema.org
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION
LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.name="splitfuse" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/SchnorcherSepp/splitfuse/" \
      org.label-schema.version=$VERSION \
      org.label-schema.schema-version="1.0"

# install rclone and splitfuse
RUN apk add --no-cache fuse go git musl-dev && \
    sed -i "s/#user_allow_other/user_allow_other/g" /etc/fuse.conf && \
    go get github.com/SchnorcherSepp/splitfuse && \
    go build -o /usr/bin/splitfuse github.com/SchnorcherSepp/splitfuse && \
    go get github.com/ncw/rclone && \
    go build -o /usr/bin/rclone github.com/ncw/rclone && \
    apk del go git musl-dev && \
    rm -R /root/go/ && \
    splitfuse --version && \
    rclone --version

# TODO
VOLUME ["/config"]

ENTRYPOINT ["/usr/bin/rclone"]

CMD ["--version"]
