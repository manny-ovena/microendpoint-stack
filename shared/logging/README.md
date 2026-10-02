# Shared logging

This package creates structured JSON logs with the core fields specified in
[`spec/observability.md`](../../spec/observability.md): `timestamp`,
`service_name`, `service_version`, `environment`, optional `pod_name` and
`node_name`, and `log_level`.

```go
logger, err := logging.NewFromEnv("endpoint-users-get")
if err != nil {
    return err
}
logger.Info().Str("http_method", "GET").Msg("request complete")
```

`New` accepts a `Config` for explicit metadata and an optional output writer.
Otherwise it reads `SERVICE_VERSION` (or `GIT_SHA`), `ENVIRONMENT`, `POD_NAME`,
`NODE_NAME`, and `LOG_LEVEL` from the environment. Version and environment
default to `dev`, and log level defaults to `info`. `WithTraceContext` adds
non-empty `trace_id`, `span_id`, and `correlation_id` fields to a logger.

Fields specific to an endpoint, subscriber, or broker are added at the call
site with zerolog's typed field methods.
