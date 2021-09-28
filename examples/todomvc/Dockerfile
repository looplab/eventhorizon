FROM golang:1.17-alpine3.14 as builder

RUN apk -U upgrade && \
    apk add --update ca-certificates tzdata curl gzip

RUN curl -L -o elm.gz https://github.com/elm/compiler/releases/download/0.19.1/binary-for-linux-64-bit.gz && \
    gunzip elm.gz && \
    chmod +x elm && \
    mv elm /usr/local/bin/

WORKDIR /eventhorizon
COPY go.mod go.mod
RUN go mod download
COPY . .

# Build frontend.
WORKDIR /eventhorizon/examples/todomvc/frontend
RUN elm make src/Main.elm --output=elm.js

# Build backend.
WORKDIR /eventhorizon/examples/todomvc/backend
RUN CGO_ENABLED=0 go build .

FROM alpine:3.14

# Import certs and timezone data.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /eventhorizon/examples/todomvc/frontend/index.html frontend/
COPY --from=builder /eventhorizon/examples/todomvc/frontend/elm.js frontend/
COPY --from=builder /eventhorizon/examples/todomvc/frontend/css/* frontend/css/

COPY --from=builder /eventhorizon/examples/todomvc/backend/backend .

ENTRYPOINT ["/backend"]
