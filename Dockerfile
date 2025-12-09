# ---------- Builder ----------
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

# Set working directory for Go build
WORKDIR /app

# Copy Go dependency files from service folder
COPY go.mod go.sum ./

RUN go mod download
RUN go mod verify

# Copy service source code
#COPY services/postService/. .
COPY . .
# Build the Go app
RUN CGO_ENABLED=1 go build -tags dynamic -ldflags="-w -s" -o app ./cmd/server

# ---------- Final Image ----------
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates nano && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# Copy SSL certs from nginx folder
#COPY nginx/certs/ca.crt /usr/local/share/ca-certificates/ca.crt
#COPY nginx/certs/ca.key /etc/ssl/private/ca.key

# Set permissions and update CA store
RUN #chmod 600 /etc/ssl/private/ca.key && update-ca-certificates

# Copy built artifacts and libraries from builder
COPY --from=builder /usr/local/lib/librdkafka* /usr/local/lib/
COPY --from=builder /app/app .
COPY --from=builder /app/config ./config
COPY --from=builder /app/internal/bootstrap ./internal

# Configure environment
ENV LD_LIBRARY_PATH=/usr/local/lib

EXPOSE 8080
CMD ["./app"]