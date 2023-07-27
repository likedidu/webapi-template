FROM golang:1.20.3-alpine as builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -ldflags="-w -s" -o /app/webapi-template .

FROM scratch

WORKDIR /app

COPY --from=builder /app/webapi-template .

EXPOSE 3000

CMD [ "./webapi-template", "0.0.0.0:3000" ]
