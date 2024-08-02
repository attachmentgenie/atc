FROM scratch
ENTRYPOINT ["/atc", "server"]
COPY atc /