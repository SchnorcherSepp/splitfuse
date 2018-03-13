FROM debian:latest

COPY ./install.sh /tmp/
COPY ./setup.sh /tmp/

RUN chmod +x /tmp/*.sh \
 && /tmp/install.sh \
 && /tmp/setup.sh
