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
