# Changes

### 2016-09-18

AWS DynamoDB event store is now in master. Also the Wercker CI config was updated to use the new Docker pipeline.

### 2016-06-17

Added exprimental support for an event store using AWS DynamoDB in the branch "dynamodb".


### 2016-06-16

Use native Go testing instead of Go check everywhere.


### 2016-06-10

Use native Go testing instead of Go check for the core types. Fix a small, unlikely bug where the aggregate could be nil in the command handler.


### 2016-06-09

Move storage and messaging implementations to its own packages for better dependency management and readability.

Move common domain objects to its own package for the examples.


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