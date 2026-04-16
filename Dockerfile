From golang:1.25-alpine as Builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o /main


FROM alpine:latest

COPY --from=builder /main /main
COPY --from=builder /app/.env .env

EXPOSE 9333

CMD ["./main"]