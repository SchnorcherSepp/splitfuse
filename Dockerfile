FROM debian:latest

COPY *.sh /tmp/

RUN apt update && apt install -y unattended-upgrades && apt dist-upgrade -y
RUN chmod +x /tmp/*.sh && /tmp/install.sh && /tmp/setup.sh
