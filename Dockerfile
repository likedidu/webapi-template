FROM golang:latest

ENV PORT 8080

WORKDIR /app

COPY . .

RUN go build -o main .

ENTRYPOINT ["./main", "0.0.0.0:8080"]

EXPOSE 8080
