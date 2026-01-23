FROM golang:1.22.6 AS build

WORKDIR /app

# this is necessary for the replace in go.mod
COPY ../../ .

WORKDIR /app/e2e/custom-headers

RUN go mod download
RUN go build -o /custom-headers .

FROM golang:1.22.6

WORKDIR /app

COPY --from=build /custom-headers /app/custom-headers

EXPOSE 8080

CMD [ "/app/custom-headers" ]
