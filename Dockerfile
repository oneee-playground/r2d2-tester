FROM golang:1.22 AS build

WORKDIR /build

COPY ./cmd ./cmd
COPY ./internal ./internal
COPY go.mod go.mod
COPY go.sum go.sum 

RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/app


FROM alpine:latest

COPY --from=build /build/app ./app

ENTRYPOINT [ "./app" ]