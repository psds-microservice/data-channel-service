# No replace, no go.work: inject infra into GOMODCACHE.
FROM golang:1.26-alpine AS builder
ENV GOMODCACHE=/go/pkg/mod
WORKDIR /app
COPY infra /go/pkg/mod/github.com/psds-microservice/infra@v0.0.3
COPY data-channel-service ./data-channel-service
WORKDIR /app/data-channel-service
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /data-channel-service ./cmd/data-channel-service

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /data-channel-service .
COPY --from=builder /app/data-channel-service/database ./database
EXPOSE 8093
CMD ["./data-channel-service", "api"]
