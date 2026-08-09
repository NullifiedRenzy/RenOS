FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /renos-core ./cmd/renos-core/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /renos-core /usr/local/bin/renos-core
EXPOSE 8443 8444
ENTRYPOINT ["renos-core"]
CMD ["migrate"]
