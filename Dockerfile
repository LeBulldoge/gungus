FROM alpine:3.18
WORKDIR /app

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

ARG USERNAME=gungus
ARG UID=65536
ARG GID=$UID
RUN addgroup -g "$GID" "$USERNAME" \
    && adduser -S -u "$UID" -G "$USERNAME" "$USERNAME"

COPY ./build/gungus.$TARGETOS.$TARGETARCH ./gungus

USER $UID
ENTRYPOINT ["./gungus", "-config", "/config"]
