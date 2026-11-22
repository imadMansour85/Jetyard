FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Set by the release pipeline; GET /version reports them.
ARG VERSION=dev
ARG COMMIT=""
ARG BUILD_TIME=""
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
      -X github.com/imadMansour85/Jetyard/backend/buildinfo.Version=${VERSION} \
      -X github.com/imadMansour85/Jetyard/backend/buildinfo.Commit=${COMMIT} \
      -X github.com/imadMansour85/Jetyard/backend/buildinfo.BuildTime=${BUILD_TIME}" \
    -o jetyard .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/jetyard .

EXPOSE 8080

CMD ["./jetyard"]
