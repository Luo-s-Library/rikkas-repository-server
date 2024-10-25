FROM golang:1.23.2-alpine AS build

WORKDIR /app

COPY ./src .
COPY ~/.aws/credentials .

RUN go build -o RikkasRepositoryServer

FROM alpine:latest

WORKDIR /app

COPY --from=build /app/build/ /app/
RUN mkdir -p /root/.aws
COPY --from=build /app/credentials /root/.aws/credentials
COPY --from=build /app/RikkasRepositoryServer /app/RikkasRepositoryServer

EXPOSE 8080

CMD ["./RikkasRepositoryServer"]