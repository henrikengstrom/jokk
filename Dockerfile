FROM alpine:latest

RUN mkdir -p /opt/program

COPY jokk /opt/program
COPY jokk.toml /opt/program

RUN chgrp -R root /opt/program && \
    chmod g+rw -R /opt/program

ENTRYPOINT ["/opt/program/jokk", "-c", "/opt/program/jokk.toml", "-n", "local", "interactiveMode"]
