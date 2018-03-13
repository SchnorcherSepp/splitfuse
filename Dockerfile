FROM debian:latest

COPY setup.sh /tmp/

RUN chmod +x /tmp/setup.sh
RUN /tmp/setup.sh
