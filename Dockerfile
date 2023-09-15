FROM golang:1.21-alpine as builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY main.go ./
RUN go build -ldflags="-w -s" -trimpath

FROM alpine:3.18
WORKDIR /app

ARG USERNAME=gungus
ARG UID=65536
ARG GID=$UID
RUN addgroup -g "$GID" "$USERNAME" \
    && adduser -S -u "$UID" -G "$USERNAME" "$USERNAME"

COPY --from=builder /app/gungus ./

USER $UID
ENTRYPOINT ["./gungus"]
