FROM debian:latest

COPY *.sh /tmp/
RUN chmod +x /tmp/*.sh

RUN /tmp/install.sh
RUN /tmp/setup.sh
