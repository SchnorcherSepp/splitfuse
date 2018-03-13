FROM debian:latest

COPY setup.sh /tmp/
COPY install.sh /tmp/

RUN chmod +x /tmp/install.sh
RUN chmod +x /tmp/setup.sh
RUN /tmp/install.sh
RUN /tmp/setup.sh
