FROM debian:latest

ADD . /tmp

RUN cd /tmp
&&  chmod +x ./setup.sh
&&  ./setup.sh
