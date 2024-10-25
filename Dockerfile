FROM golang:1.23.1-alpine AS build

WORKDIR /src

ARG AWS_CREDENTIALS_PATH
COPY src/ /src/
COPY  ${AWS_CREDENTIALS_PATH} /root/.aws/credentials

RUN go build -o /src/build/RikkasRepositoryServer .

FROM alpine:latest

WORKDIR /app

COPY --from=build /src/build/ /app/
COPY --from=build /root/.aws/credentials /root/.aws/credentials

EXPOSE 80

CMD ["./RikkasRepositoryServer"]