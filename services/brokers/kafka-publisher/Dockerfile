FROM golang:1.27 AS builder

ENV GOWORK=off
WORKDIR /workspace

COPY services/brokers/kafka-publisher/go.mod services/brokers/kafka-publisher/go.sum ./services/brokers/kafka-publisher/
RUN cd services/brokers/kafka-publisher && go mod download

COPY services/brokers/kafka-publisher/ ./services/brokers/kafka-publisher/
RUN cd services/brokers/kafka-publisher && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /kafka-publisher ./cmd/server

FROM scratch

COPY --from=builder /kafka-publisher /kafka-publisher
USER 65532:65532
EXPOSE 9000
ENTRYPOINT ["/kafka-publisher"]
