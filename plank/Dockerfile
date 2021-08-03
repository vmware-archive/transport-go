FROM photon:latest
COPY build/eve-platform /usr/local/bin/
RUN mkdir -p /opt/eve-platform
WORKDIR /opt/eve-platform
STOPSIGNAL SIGTERM
ENTRYPOINT ["eve-platform", "start-server"]
