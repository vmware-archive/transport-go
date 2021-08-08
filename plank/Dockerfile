FROM photon:latest
COPY build/plank /usr/local/bin/
RUN mkdir -p /opt/plank
WORKDIR /opt/plank
STOPSIGNAL SIGTERM
ENTRYPOINT ["plank", "start-server"]
