FROM golang:latest

ENV PORT 3000

WORKDIR /app

COPY . .

RUN go build -o main .

ENTRYPOINT ["./main", "0.0.0.0:3000"]

EXPOSE 3000
