# Builder
FROM golang:1.23-alpine AS builder

ARG VERSION=dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o beacon ./cmd/beacon

# The beacon-dns image
FROM scratch

COPY --from=builder /app/beacon /beacon

EXPOSE 80
EXPOSE 53

CMD ["/beacon"]
