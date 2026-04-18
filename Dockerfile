FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/motadata-host-agent ./cmd/motadata-host-agent/

FROM gcr.io/distroless/base-debian12

COPY --from=builder /app/motadata-host-agent /motadata-host-agent

USER nonroot:nonroot

EXPOSE 8181

ENTRYPOINT ["/motadata-host-agent"]
