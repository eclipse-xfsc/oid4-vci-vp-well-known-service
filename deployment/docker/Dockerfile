FROM golang:1.21.5-alpine3.19 as build

WORKDIR /app

ADD . .

RUN CGO_ENABLED=0 GOOS=linux go build

FROM alpine:3.15 as runner

COPY --from=build /app/well-known-service /opt/well-known-service

WORKDIR /opt

EXPOSE 8080

CMD ["./well-known-service"]
