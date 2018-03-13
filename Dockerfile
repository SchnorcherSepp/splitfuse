FROM debian:latest

ADD . /tmp

RUN cd /tmp
RUN chmod +x ./setup.sh
RUN ./setup.sh
