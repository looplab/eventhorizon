# TodoMVC with Event Horizon

This is a full example of using Event Horizon, including a frontend in Elm. The Elm frontend is based on the official Elm port of TodoMVC: https://github.com/evancz/elm-todomvc.

## Usage

Run the Elm frontend in the reactor:
```bash
cd ui && elm-reactor
```

Or compile the Elm frontend:
```bash
cd ui && elm-make Main.elm --output elm.js
```

Run MongoDB in Docker:
```bash
docker run -d --name mongo -p 27017:27017 mongo:latest
```

Run the Event Horizon backend, which also serves the frontend:
```bash
go run main.go handler.go
```

Visit http://localhost:8080

To run the tests:
```bash
go test ./...
```
