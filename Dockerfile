FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o myapp .

FROM debian:bullseye-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        gnupg && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

ADD https://repository.certum.pl/ovcasha2.pem /usr/local/share/ca-certificates/duw-intermediate.crt
RUN update-ca-certificates

COPY --from=builder /app/myapp .

CMD ["./myapp"]
