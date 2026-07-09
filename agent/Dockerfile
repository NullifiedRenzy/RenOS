FROM rust:1.88-alpine AS builder

RUN apk add --no-cache musl-dev protoc
WORKDIR /src/agent
COPY agent .
COPY proto /src/proto
RUN cargo build --release --bin renos-agent

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /src/agent/target/release/renos-agent /usr/local/bin/renos-agent
ENTRYPOINT ["renos-agent"]
CMD ["--help"]
