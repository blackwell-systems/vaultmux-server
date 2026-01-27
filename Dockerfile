FROM golang:1.24-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" \
    -o vaultmux-server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /build/vaultmux-server /vaultmux-server

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/vaultmux-server"]
