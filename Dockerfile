# Builder
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o beacon ./cmd/beacon

# The beacon-dns image
FROM scratch

COPY --from=builder /app/beacon /beacon

EXPOSE 8080
EXPOSE 2053
EXPOSE 2853

CMD ["/beacon"]
