FROM golang:1.24 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY client/ ./client/
COPY db/ ./db/
COPY handlers/ ./handlers/
COPY models/ ./models/
COPY temporal/ ./temporal/

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gin-service .

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache libc6-compat
COPY --from=build /app/gin-service .
EXPOSE 8081
CMD ["./gin-service"]