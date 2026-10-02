FROM golang:1.27 AS builder

ENV GOWORK=off
WORKDIR /workspace/services/endpoints/inventory.adjust

COPY shared/logging/go.mod shared/logging/go.sum /workspace/shared/logging/
COPY services/endpoints/inventory.adjust/go.mod services/endpoints/inventory.adjust/go.sum ./
RUN go mod download

COPY shared/logging/ /workspace/shared/logging/
COPY services/endpoints/inventory.adjust/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /inventory-adjust ./cmd

FROM scratch

COPY --from=builder /inventory-adjust /inventory-adjust
USER 65532:65532
ENTRYPOINT ["/inventory-adjust"]
