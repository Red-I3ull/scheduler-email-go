FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /go-scheduler main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /go-scheduler .
CMD ["./go-scheduler"]