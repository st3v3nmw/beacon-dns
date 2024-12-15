# Builder
FROM golang:1.23-alpine AS builder

ARG VERSION=dev
ARG BUILD_DATE
ARG COMMIT_REF

LABEL\
	maintainer="Stephen Mwangi <mail@stephenmwangi.com>" \
	org.opencontainers.image.authors="Stephen Mwangi <mail@stephenmwangi.com>" \
	org.opencontainers.image.created=$BUILD_DATE \
	org.opencontainers.image.description="A DNS resolver with customizable & schedulable filtering for malware, trackers, ads, and other unwanted content" \
	org.opencontainers.image.documentation="https://www.beacondns.org" \
	org.opencontainers.image.licenses="MIT" \
	org.opencontainers.image.revision=$COMMIT_REF \
	org.opencontainers.image.source="https://github.com/st3v3nmw/beacon-dns" \
	org.opencontainers.image.title="Beacon DNS" \
	org.opencontainers.image.url="https://www.beacondns.org" \
	org.opencontainers.image.vendor="Beacon DNS" \
	org.opencontainers.image.version=$VERSION

ENV CGO_ENABLED=1

WORKDIR /app

RUN apk add --no-cache gcc musl-dev wget unzip

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN wget https://github.com/nalgeon/sqlean/releases/download/0.27.1/sqlean-linux-x86.zip -O /tmp/sqlean.zip \
    && unzip /tmp/sqlean.zip -d ./sqlean

RUN go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o beacon ./cmd/beacon

# The beacon-dns image
FROM alpine:latest

RUN apk add --no-cache libc6-compat tzdata

COPY --from=builder /app/beacon /beacon
COPY --from=builder /app/sqlean/stats.so /extensions/stats.so

EXPOSE 80
EXPOSE 53

CMD ["/beacon"]
