FROM golang:1.25-alpine AS builder

ARG CHECKER_VERSION=custom-build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.Version=${CHECKER_VERSION}" -o /checker-zonemaster .

FROM scratch
COPY --from=builder /checker-zonemaster /checker-zonemaster
USER 65534:65534
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/checker-zonemaster", "-healthcheck"]
ENTRYPOINT ["/checker-zonemaster"]
