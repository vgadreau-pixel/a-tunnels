FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o atunnels-server ./cmd/server

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/atunnels-server /app/
COPY atunnels.yml /app/atunnels.yml

EXPOSE 80 443 8080 2222 27200 10000-10100

ENTRYPOINT ["/app/atunnels-server"]
CMD ["--config", "/app/atunnels.yml"]
