FROM ubuntu:latest
LABEL authors="deck"

ENTRYPOINT ["top", "-b"]