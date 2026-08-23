From golang:1.27-alpine as builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o /main


FROM alpine:latest

COPY --from=builder /main /main

EXPOSE 9333

CMD ["./main"]