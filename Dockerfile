FROM debian:latest

COPY install.sh /tmp/
COPY setup.sh /tmp/

RUN chmod +x /tmp/install.sh \
 && chmod +x /tmp/setup.sh \
 && /tmp/install.sh \
 && /tmp/setup.sh
