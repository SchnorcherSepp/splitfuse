FROM debian:latest

COPY *.sh /tmp/

RUN chmod +x /tmp/*.sh && /tmp/install.sh
RUN /tmp/setup.sh
