FROM golang:1.24.0 AS build
WORKDIR /app
COPY . .
WORKDIR /app/e2e/metrics
RUN go mod download
RUN go build -o /metrics

FROM golang:1.24.0
WORKDIR /app
COPY --from=build /metrics /app/metrics
EXPOSE 8081
CMD [ "/app/metrics" ]
