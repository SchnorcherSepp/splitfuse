FROM debian:latest

COPY setup.sh /tmp/
COPY install.sh /tmp/

RUN chmod +x /tmp/install.sh \
 && chmod +x /tmp/setup.sh \
 && /tmp/install.sh \
 && /tmp/setup.sh
