FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o gin-service .

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/gin-service .
EXPOSE 8081
CMD ["./gin-service"]
