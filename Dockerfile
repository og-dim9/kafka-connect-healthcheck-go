FROM golang:1.21.13-alpine AS builder
COPY cmd /src/cmd/
WORKDIR /src/cmd/healthcheck
RUN go build -o healthcheck --ldflags "-s -w" .

FROM scratch as final
WORKDIR /
COPY README.md LICENSE
COPY --from=builder /src/cmd/healthcheck/healthcheck /bin/healthcheck

EXPOSE 18083
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 CMD curl -f http://localhost:18083/healthz || exit 1
CMD ["/bin/healthcheck"]


