# TodoMVC with Event Horizon

This is a full example of using Event Horizon, including a frontend in Elm. The Elm frontend is based on the official Elm port of TodoMVC: https://github.com/evancz/elm-todomvc.

## Usage

Run the backend together with MongoDB, this will also compile the frontend:

```bash
make run
```

Visit http://localhost:8080

To run the tests (requires that MongoDB is runnng):

```bash
go test ./...
```

To stop MongoDB:

```bash
make stop
```
