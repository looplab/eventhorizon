# TodoMVC with Event Horizon

This is a full example of using Event Horizon, including a frontend in Elm. The Elm frontend is based on the official Elm port of TodoMVC: https://github.com/evancz/elm-todomvc.

## Usage

To run the example with Docker, which will also compile it:

```bash
make run
```

Visit http://localhost:8080 for the TodoMVC app and http://localhost:16686 to view the traces.

Or to run the example locally (requires Elm to be installed):

```bash
make build_frontend run_services run_backend
```

To run the tests (requires `make run_services`):

```bash
go test ./...
```

To stop the services:

```bash
make stop
```
