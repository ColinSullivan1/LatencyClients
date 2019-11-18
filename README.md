# Latency Clients

This includes a simple Prometheus enabled requestor, a replier, 
and tooling to build images and test.

## Building

You can build executables locally for testing:

`make build` or `make`

`make clean` cleans up.

## Building Images

Build images with `make images`.

## Pushing Images

Push images to dockerhub with `make push`.


## Testing with docker compose

1) Edit the .env file as necessary
2) `cd docker-compose`
3) `docker-compose up`