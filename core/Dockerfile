FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /renos-core ./cmd/renos-core/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /renos-core /usr/local/bin/renos-core
EXPOSE 8443 8444
ENTRYPOINT ["renos-core"]
CMD ["migrate"]
