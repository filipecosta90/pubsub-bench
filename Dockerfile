FROM golang:1.18 as builder
WORKDIR /work
ADD . .
RUN CGO_ENABLED=0 go build -o /pubsub-bench .

# runner
FROM golang:1.18-alpine
WORKDIR /
COPY --from=builder /pubsub-bench /bin/pubsub-bench

CMD ["/bin/pubsub-bench"]