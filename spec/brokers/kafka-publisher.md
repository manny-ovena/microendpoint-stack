# Kafka Publishing Broker Specification

## Overview

The Kafka Publishing Broker is an application-facing service for producing
messages to the Kafka cluster. It centralizes Kafka producer connections so
HTTP endpoint replicas do not each own a Kafka producer. It is separate from
the Kafka cluster itself, which is deployed in `k8s/base/kafka.yaml`.

## RPC Method

### Publish

Accepts a topic, optional key, payload, optional headers, and optional timeout.
It waits for Kafka to acknowledge the message from all in-sync replicas before
returning the topic and an acknowledged status. Failures are returned as
errors. The effective durability depends on the topic's replication factor.

## Connection Management

- Reuse long-lived producer clients within each service process.
- Configure bootstrap addresses with `KAFKA_BROKERS`, a comma-separated list.
- Close the producer during service shutdown.
- Bound the Kafka-facing connection count by configuring a maximum replica
  count for the publishing service.
- Endpoint replicas may scale independently; they connect to the publishing
  service rather than directly creating Kafka producers.

The implementation uses one synchronous `kafka-go` writer per service
instance and a MessagePack TCP RPC server. Kubernetes deployment manifests
remain separate work.
