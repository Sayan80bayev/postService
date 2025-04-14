FROM golang:1.24-bullseye AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    gcc g++ make curl pkg-config git

# Install librdkafka from source
RUN git clone https://github.com/confluentinc/librdkafka.git && \
    cd librdkafka && \
    git checkout v1.9.0 && \
    ./configure && \
    make && \
    make install

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .
RUN CGO_ENABLED=1 go build -tags dynamic -ldflags="-w -s" -o app ./cmd/server

# Final slim image
FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y ca-certificates &&  apt-get install -y nano &&\
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /usr/local/lib/librdkafka* /usr/local/lib/
COPY --from=builder /app/app .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/internal/bootstrap ./internal

ENV LD_LIBRARY_PATH=/usr/local/lib

EXPOSE 8080
CMD ["./app"]