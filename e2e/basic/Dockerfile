FROM golang:1.19 AS build

WORKDIR /app

# this is necessary for the replace in go.mod
COPY ../../ .

WORKDIR /app/e2e/basic

RUN go mod download
RUN go build -o /basic

FROM golang:1.19

WORKDIR /app

COPY --from=build /basic /app/basic

EXPOSE 8080

CMD [ "/app/basic" ]