[![PkgGoDev](https://pkg.go.dev/badge/github.com/looplab/eventhorizon)](https://pkg.go.dev/github.com/looplab/eventhorizon)
![Bulid Status](https://github.com/looplab/eventhorizon/actions/workflows/main.yml/badge.svg)
[![Coverage Status](https://img.shields.io/coveralls/looplab/eventhorizon.svg)](https://coveralls.io/r/looplab/eventhorizon)
[![Go Report Card](https://goreportcard.com/badge/looplab/eventhorizon)](https://goreportcard.com/report/looplab/eventhorizon)

# Event Horizon

Event Horizon is a CQRS/ES toolkit for Go.

**NOTE: Event Horizon is used in production systems but the API is not final!**

CQRS stands for Command Query Responsibility Segregation and is a technique where object access (the Query part) and modification (the Command part) are separated from each other. This helps in designing complex data models where the actions can be totally independent from the data output.

ES stands for Event Sourcing and is a technique where all events that have happened in a system are recorded, and all future actions are based on the events instead of a single data model. The main benefit of adding Event Sourcing is traceability of changes which can be used for example in audit logging. Additionally, "incorrect" events that happened in the past (for example due to a bug) can be compensated for with an event which will make the current data "correct", as that is based on the events.

Read more about CQRS/ES from one of the major authors/contributors on the subject: http://codebetter.com/gregyoung/2010/02/16/cqrs-task-based-uis-event-sourcing-agh/

Other material on CQRS/ES:

- http://martinfowler.com/bliki/CQRS.html
- http://cqrs.nu
- https://groups.google.com/forum/#!forum/dddcqrs

Inspired by the following libraries/examples:

- https://github.com/edumentab/cqrs-starter-kit
- https://github.com/pjvds/go-cqrs
- http://www.codeproject.com/Articles/555855/Introduction-to-CQRS
- https://github.com/qandidate-labs/broadway

Suggestions are welcome!

# Usage

See the example folder for a few examples to get you started.

# Get Involved

- Join our [slack channel](https://gophers.slack.com/messages/eventhorizon/) (sign up [here](https://gophersinvite.herokuapp.com/))
- Check out the [contribution guidelines](CONTRIBUTING.md)

# Event Store Implementations

### Official

- Memory - Useful for testing and experimentation.
- MongoDB - One document per aggregate with events as an array. Beware of the 16MB document size limit that can affect large aggregates.
- MongoDB v2 - One document per event with an additional document per aggregate. This event store is also capable of keeping track of the global event position, in addition to the aggregate version.
- Recorder - An event recorder (middleware) that can be used in tests to capture some events.
- Tracing - Adds distributed tracing support to event store operations with OpenTracing.

### Contributions / 3rd party

- AWS DynamoDB: https://github.com/seedboxtech/eh-dynamo
- Postgress: https://github.com/giautm/eh-pg
- Redis: https://github.com/TerraSkye/eh-redis

# Event Bus Implementations

### Official

- GCP Cloud Pub/Sub - Using one topic with multiple subscribers.
- NATS - Using Jetstream features.
- Kafka - Using one topic with multiple consumer groups.
- Local - Useful for testing and experimentation.
- Redis - Using Redis streams.
- Tracing - Adds distributed tracing support to event publishing and handling with OpenTracing.

### Contributions / 3rd party

- Kafka: https://github.com/Kistler-Group/eh-kafka
- NATS Streaming: https://github.com/v0id3r/eh-nats

# Repo Implementations

### Official

- Memory - Useful for testing and experimentation.
- MongoDB - One document per projected entity.
- Version - Adds support for reading a specific version of an entity from an underlying repo.
- Cache - Adds support for in-memory caching of entities from an underlying repo.
- Tracing - Adds distributed tracing support to an repo operations with OpenTracing.

# Development

To develop Event Horizon you need to have Docker and Docker Compose installed.

To run all unit tests:

```bash
make test
```

To run and stop services for integration tests:

```bash
make run
make stop
```

To run all integration tests:

```bash
make test_integration
```

Testing can also be done in docker:

```bash
make test_docker
make test_integration_docker
```

# License

Event Horizon is licensed under Apache License 2.0

http://www.apache.org/licenses/LICENSE-2.0
