FROM golang:1.27 AS builder

WORKDIR /workspace

COPY shared/methodingress-controller/go.mod shared/methodingress-controller/go.sum ./shared/methodingress-controller/
COPY shared/logging/go.mod shared/logging/go.sum ./shared/logging/
# Use only the modules needed by this image, independent of the host workspace.
RUN go work init ./shared/methodingress-controller ./shared/logging \
    && go -C shared/methodingress-controller mod download \
    && go -C shared/logging mod download

COPY shared/methodingress-controller/ ./shared/methodingress-controller/
COPY shared/logging/ ./shared/logging/

RUN CGO_ENABLED=0 go build -trimpath -o /manager ./shared/methodingress-controller/cmd

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /manager /manager

ENTRYPOINT ["/manager"]
