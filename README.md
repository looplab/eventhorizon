[![wercker status](https://app.wercker.com/status/a696ceaee4d3fc70f9cc505d002d633d/s "wercker status")](https://app.wercker.com/project/bykey/a696ceaee4d3fc70f9cc505d002d633d)
[![Coverage Status](https://img.shields.io/coveralls/looplab/eventhorizon.svg)](https://coveralls.io/r/looplab/eventhorizon)
[![GoDoc](https://godoc.org/github.com/looplab/eventhorizon?status.svg)](https://godoc.org/github.com/looplab/eventhorizon)
[![Go Report Card](https://goreportcard.com/badge/looplab/eventhorizon)](https://goreportcard.com/report/looplab/eventhorizon)

# Event Horizon

Event Horizon is a CQRS/ES toolkit for Go.

**NOTE: Event Horizon is used in production systems but the API is not final!**

CQRS stands for Command Query Responsibility Segregation and is a technique where object access (the Query part) and modification (the Command part) are separated from each other. This helps in designing complex data models where the actions can be totally independent from the data output.

ES stands for Event Sourcing and is a technique where all events that have happened in a system are recorded, and all future actions are based on the events instead of a single data model. The main benefit of adding Event Sourcing is traceability of changes which can be used for example in audit logging. Additionally, "incorrect" events that happened in the past (for example due to a bug) can be compensated for with an event which will make the current data "correct", as that is based on the events.

Read more about CQRS/ES from one of the major authors/contributors on the subject: http://codebetter.com/gregyoung/2010/02/16/cqrs-task-based-uis-event-sourcing-agh/

Other material on CQRS/ES:

* http://martinfowler.com/bliki/CQRS.html
* http://cqrs.nu
* https://groups.google.com/forum/#!forum/dddcqrs

Inspired by the following libraries/examples:

* https://github.com/edumentab/cqrs-starter-kit
* https://github.com/pjvds/go-cqrs
* http://www.codeproject.com/Articles/555855/Introduction-to-CQRS
* https://github.com/qandidate-labs/broadway

Suggestions are welcome!

# Usage

See the example folder for a few examples to get you started.

# Storage drivers

These are the drivers for storage of events and entities.

### Local / in memory

There are simple in memory implementations of an event store and entity repo. These are meant for testing/experimentation.

### MongoDB

Fairly mature, used in production.

### AWS DynamoDB

Experimental support for AWS DynamoDB as an event store. Not actively developed.

# Messaging drivers

These are the drivers for messaging, currently only publishers.

### Local / in memory

Fully synchrounos. Useful for testing/experimentation.

### Redis

Fairly mature, used in production.

### GCP Cloud Pub/Sub

Experimental driver.

# Get Involved

* Join our [slack channel](https://gophers.slack.com/messages/eventhorizon/) (sign up [here](https://gophersinvite.herokuapp.com/))
* Check out the [contribution guidelines](CONTRIBUTING.md)

# License

Event Horizon is licensed under Apache License 2.0

http://www.apache.org/licenses/LICENSE-2.0
