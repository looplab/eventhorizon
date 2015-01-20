[![wercker status](https://app.wercker.com/status/a696ceaee4d3fc70f9cc505d002d633d/s "wercker status")](https://app.wercker.com/project/bykey/a696ceaee4d3fc70f9cc505d002d633d)
[![Coverage Status](https://img.shields.io/coveralls/looplab/eventhorizon.svg)](https://coveralls.io/r/looplab/eventhorizon)
[![GoDoc](https://godoc.org/github.com/looplab/eventhorizon?status.svg)](https://godoc.org/github.com/looplab/eventhorizon)


# Event Horizon

Event Horizon is a CQRS/ES toolkit for Go.

**Event Horizon is currently under heavy development and is not considered to be stable!**

CQRS stands for Command Query Responsibility Segregation and is a technique where object access (the Query part) and modification (the Command part) are separated from each other. This helps in designing complex data models where the actions can be totally independent from the data output.

ES stands for Event Sourcing and is a technique where all events that have happened in a system are recorded, and all future actions are based on the events instead of a single data model. The main benefit of adding Event Sourcing is tracability of changes which can be used for example in audit logging. Additionally, "incorrect" events that happened in the past (for example due to a bug) can be edited which will make the current data "correct", as that is based on the events.

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

See the example folder for a basic usage example to get you started.


# Changes

### 2015-01-20

Addded CommandBus that routes commands to handlers. This is for upcoming Saga support. The dispatcher is now renamed to AggregateCommandHandler and must be added to the CommandBus. At the moment Commands have to registered both in the handler and on the bus, this may change in the future.

Added MongoDB ReadRepository implementation. Use with "-tags mongo", same as the MongoDB event store.


### 2015-01-14

Added Repository that creates/loads and saves aggregates. This needed additional methods in the Aggregate interface.

Removed the reflection based dispatcher, the code was worse performing and harder to test. There was also a bit too much magic going on. If you would like it back open an issue for further discussion.

Renamed Repository to ReadRepository to better adhere to CQRS standards and to free the name to a Aggregate/Saga repository in development.


### 2015-01-12

Added an EventStore implementation for MongoDB. It currently uses one document per aggregate with all events as an array to make the most out of MongoDBs lack of trasactions. It still takes two operations when adding events but at least there is a check that the version has not been changed by another operation in between. If you want to use the MongoDB event store add "-tags mongo" to your project build.


### 2015-01-07

As of this version commands and events are recommended to be passed around as pointers, instead of values as the previous versions did. Passing as values may still work, but is not tested at the momemnt. It should not requrie much changes in applications using Event Horizon, simple pass all commands and events with & before them or create them as *XXXCommand, see the examples and tests for usage. There are also some other API changes to method names, mostly with using "handler" as a more common term.


# License

Event Horizon is licensed under Apache License 2.0

http://www.apache.org/licenses/LICENSE-2.0
