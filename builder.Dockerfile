ARG BUILDPLATFORM
FROM --platform=${BUILDPLATFORM} golang:1.21-alpine as builder

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -trimpath

FROM --platform=${TARGETPLATFORM} alpine:3.18
WORKDIR /app

ARG USERNAME=gungus
ARG UID=65536
ARG GID=$UID
RUN addgroup -g "$GID" "$USERNAME" \
    && adduser -S -u "$UID" -G "$USERNAME" "$USERNAME"

COPY --from=builder /app/gungus ./

USER $UID
ENTRYPOINT ["./gungus", "-config", "/config"]
