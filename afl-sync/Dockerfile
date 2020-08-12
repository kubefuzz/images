# Go build container
FROM golang:1.13-alpine3.11 as build

WORKDIR /go/src/afl-sync

# Download modules (doing this separately makes it cacheabel by Docker)
COPY go.mod go.sum ./
RUN go mod download

# Compile packages and dependencies
COPY main.go ./
RUN go build -o ./sync

# Final container
FROM alpine:3.11

COPY --from=build /go/src/afl-sync/sync /usr/bin/

CMD ["sync"]