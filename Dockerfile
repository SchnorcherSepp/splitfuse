FROM debian:latest

COPY *.sh /tmp/

RUN apt update && apt install -y unattended-upgrades && apt dist-upgrade -y && chmod +x /tmp/*.sh && /tmp/setup.sh
RUN /tmp/install.sh
