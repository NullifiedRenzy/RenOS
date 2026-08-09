# RenOS Dependency Audit Report

**Generated:** 2026-07-30T00:04:24Z
**Host:** fedora


## Go Module Dependencies


## Module: github.com/renos-project/renos/core

**Path:** `/home/ren/Projects/RenOS/core`\n

## Direct Dependencies

These are the direct (non-indirect) dependencies listed in `go.mod`:
| Dependency | Version |
|---|---|
| github.com/BurntSushi/toml | v1.6.0 |
| github.com/go-chi/chi/v5 | v5.1.0 |
| github.com/golang-jwt/jwt/v5 | v5.3.1 |
| github.com/golang-migrate/migrate/v4 | v4.18.1 |
| github.com/google/uuid | v1.6.0 |
| github.com/gorhill/cronexpr | v0.0.0-20180427100037-88b0669f7d75 |
| github.com/gorilla/websocket | v1.5.3 |
| github.com/jackc/pgx/v5 | v5.7.1 |
| github.com/nats-io/nats.go | v1.37.0 |
| github.com/prometheus/client_golang | v1.24.1 |
| github.com/redis/go-redis/v9 | v9.7.0 |
| go.opentelemetry.io/otel | v1.44.0 |
| go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp | v1.44.0 |
| go.opentelemetry.io/otel/exporters/stdout/stdouttrace | v1.44.0 |
| go.opentelemetry.io/otel/exporters/zipkin | v1.44.0 |
| go.opentelemetry.io/otel/sdk | v1.44.0 |
| go.opentelemetry.io/otel/trace | v1.44.0 |
| golang.org/x/crypto | v0.54.0 |
| golang.org/x/time | v0.15.0 |

## go mod verify

```\nall modules verified\n```

## All Dependencies (go list -m all)

```\ngithub.com/renos-project/renos/core
cel.dev/expr v0.25.1
cloud.google.com/go v0.112.1
cloud.google.com/go/compute v1.25.1
cloud.google.com/go/compute/metadata v0.9.0
cloud.google.com/go/iam v1.1.6
cloud.google.com/go/longrunning v0.5.5
cloud.google.com/go/spanner v1.56.0
cloud.google.com/go/storage v1.38.0
github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4
github.com/99designs/keyring v1.2.1
github.com/Azure/azure-sdk-for-go/sdk/azcore v1.4.0
github.com/Azure/azure-sdk-for-go/sdk/internal v1.1.2
github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.0.0
github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161
github.com/Azure/go-autorest v14.2.0+incompatible
github.com/Azure/go-autorest/autorest/adal v0.9.16
github.com/Azure/go-autorest/autorest/date v0.3.0
github.com/Azure/go-autorest/logger v0.2.1
github.com/Azure/go-autorest/tracing v0.6.0
github.com/BurntSushi/toml v1.6.0
github.com/ClickHouse/clickhouse-go v1.4.3
github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.31.0
github.com/IBM/sarama v1.43.1
github.com/Microsoft/go-winio v0.6.2
github.com/alecthomas/kingpin/v2 v2.4.0
github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b
github.com/andybalholm/brotli v1.0.4
github.com/antihax/optional v1.0.0
github.com/apache/arrow/go/v10 v10.0.1
github.com/apache/thrift v0.16.0
github.com/aws/aws-sdk-go v1.49.6
github.com/aws/aws-sdk-go-v2 v1.16.16
github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.8
github.com/aws/aws-sdk-go-v2/credentials v1.12.20
github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.11.33
github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.23
github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.17
github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.14
github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.9
github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.18
github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.17
github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.17
github.com/aws/aws-sdk-go-v2/service/s3 v1.27.11
github.com/aws/smithy-go v1.13.3
github.com/beorn7/perks v1.0.1
github.com/bsm/ginkgo/v2 v2.12.0
github.com/bsm/gomega v1.27.10
github.com/cenkalti/backoff/v4 v4.1.2
github.com/cenkalti/backoff/v5 v5.0.3
github.com/census-instrumentation/opencensus-proto v0.4.1
github.com/cespare/xxhash/v2 v2.3.0
github.com/cloudflare/golz4 v0.0.0-20150217214814-ef862a3cdc58
github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2
github.com/cockroachdb/cockroach-go/v2 v2.1.1
github.com/cznic/mathutil v0.0.0-20180504122225-ca4c9f2c1369
github.com/danieljoos/wincred v1.1.2
github.com/davecgh/go-spew v1.1.1
github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f
github.com/dhui/dktest v0.4.3
github.com/distribution/reference v0.6.0
github.com/docker/docker v27.2.0+incompatible
github.com/docker/go-connections v0.5.0
github.com/docker/go-units v0.5.0
github.com/dvsekhvalnov/jose2go v1.6.0
github.com/eapache/go-resiliency v1.6.0
github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3
github.com/eapache/queue v1.1.0
github.com/edsrzf/mmap-go v0.0.0-20170320065105-0bce6a688712
github.com/envoyproxy/go-control-plane v0.14.0
github.com/envoyproxy/go-control-plane/envoy v1.37.0
github.com/envoyproxy/go-control-plane/ratelimit v0.1.0
github.com/envoyproxy/protoc-gen-validate v1.3.3
github.com/felixge/httpsnoop v1.0.4
github.com/form3tech-oss/jwt-go v3.2.5+incompatible
github.com/fsouza/fake-gcs-server v1.17.0
github.com/gabriel-vasile/mimetype v1.4.1
github.com/go-chi/chi/v5 v5.1.0
github.com/go-jose/go-jose/v4 v4.1.4
github.com/go-logr/logr v1.4.3
github.com/go-logr/stdr v1.2.2
github.com/go-sql-driver/mysql v1.5.0
github.com/go-stack/stack v1.8.0
github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572
github.com/gobuffalo/here v0.6.0
github.com/goccy/go-json v0.9.11
github.com/gocql/gocql v0.0.0-20210515062232-b7ef815b4556
github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2
github.com/gogo/protobuf v1.3.2
github.com/golang-jwt/jwt/v4 v4.4.2
github.com/golang-jwt/jwt/v5 v5.3.1
github.com/golang-migrate/migrate/v4 v4.18.1
github.com/golang-sql/civil v0.0.0-20190719163853-cb61b32ac6fe
github.com/golang-sql/sqlexp v0.1.0
github.com/golang/glog v1.2.5
github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
github.com/golang/protobuf v1.5.4
github.com/golang/snappy v0.0.4
github.com/google/flatbuffers v2.0.8+incompatible
github.com/google/go-cmp v0.7.0
github.com/google/go-github/v39 v39.2.0
github.com/google/go-querystring v1.1.0
github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38
github.com/google/s2a-go v0.1.7
github.com/google/uuid v1.6.0
github.com/googleapis/enterprise-certificate-proxy v0.3.2
github.com/googleapis/gax-go/v2 v2.12.2
github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
github.com/gorilla/handlers v1.4.2
github.com/gorilla/mux v1.7.4
github.com/gorilla/websocket v1.5.3
github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0
github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c
github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed
github.com/hashicorp/errwrap v1.1.0
github.com/hashicorp/go-multierror v1.1.1
github.com/hashicorp/go-uuid v1.0.3
github.com/jackc/chunkreader/v2 v2.0.1
github.com/jackc/pgconn v1.14.3
github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa
github.com/jackc/pgio v1.0.0
github.com/jackc/pgpassfile v1.0.0
github.com/jackc/pgproto3/v2 v2.3.3
github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761
github.com/jackc/pgtype v1.14.0
github.com/jackc/pgx/v4 v4.18.2
github.com/jackc/pgx/v5 v5.7.1
github.com/jackc/puddle/v2 v2.2.2
github.com/jcmturner/aescts/v2 v2.0.0
github.com/jcmturner/dnsutils/v2 v2.0.0
github.com/jcmturner/gofork v1.7.6
github.com/jcmturner/gokrb5/v8 v8.4.4
github.com/jcmturner/rpc/v2 v2.0.3
github.com/jmespath/go-jmespath v0.4.0
github.com/jpillora/backoff v1.0.0
github.com/json-iterator/go v1.1.12
github.com/julienschmidt/httprouter v1.3.0
github.com/k0kubun/pp v2.3.0+incompatible
github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
github.com/klauspost/asmfmt v1.3.2
github.com/klauspost/compress v1.19.1
github.com/klauspost/cpuid/v2 v2.0.9
github.com/kr/pretty v0.3.1
github.com/kr/text v0.2.0
github.com/ktrysmt/go-bitbucket v0.6.4
github.com/kylelemons/godebug v1.1.0
github.com/lib/pq v1.10.9
github.com/markbates/pkger v0.15.1
github.com/mattn/go-colorable v0.1.6
github.com/mattn/go-isatty v0.0.16
github.com/mattn/go-sqlite3 v1.14.22
github.com/microsoft/go-mssqldb v1.0.0
github.com/minio/asm2plan9s v0.0.0-20200509001527-cdd76441f9d8
github.com/minio/c2goasm v0.0.0-20190812172519-36a3d3bbc4f3
github.com/mitchellh/mapstructure v1.1.2
github.com/moby/docker-image-spec v1.3.1
github.com/moby/term v0.5.0
github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
github.com/modern-go/reflect2 v1.0.2
github.com/morikuni/aec v1.0.0
github.com/mtibben/percent v0.2.1
github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822
github.com/mutecomm/go-sqlcipher/v4 v4.4.0
github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f
github.com/nakagami/firebirdsql v0.0.0-20190310045651-3c02a58cfed8
github.com/nats-io/nats.go v1.37.0
github.com/nats-io/nkeys v0.4.7
github.com/nats-io/nuid v1.0.1
github.com/neo4j/neo4j-go-driver v1.8.1-0.20200803113522-b626aa943eba
github.com/onsi/ginkgo v1.16.4
github.com/onsi/ginkgo/v2 v2.11.0
github.com/onsi/gomega v1.27.10
github.com/opencontainers/go-digest v1.0.0
github.com/opencontainers/image-spec v1.1.0
github.com/openzipkin/zipkin-go v0.4.3
github.com/pierrec/lz4/v4 v4.1.21
github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
github.com/pkg/errors v0.9.1
github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10
github.com/pmezard/go-difflib v1.0.0
github.com/prometheus/client_golang v1.24.1
github.com/prometheus/client_model v0.6.2
github.com/prometheus/common v0.70.1
github.com/prometheus/procfs v0.21.1
github.com/rabbitmq/amqp091-go v1.9.0
github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
github.com/redis/go-redis/v9 v9.7.0
github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0
github.com/rogpeppe/fastuuid v1.2.0
github.com/rogpeppe/go-internal v1.14.1
github.com/rqlite/gorqlite v0.0.0-20230708021416-2acd02b70b79
github.com/shopspring/decimal v1.2.0
github.com/sirupsen/logrus v1.9.3
github.com/snowflakedb/gosnowflake v1.6.19
github.com/spiffe/go-spiffe/v2 v2.6.0
github.com/stretchr/objx v0.5.2
github.com/stretchr/testify v1.11.1
github.com/xanzy/go-gitlab v0.15.0
github.com/xdg-go/pbkdf2 v1.0.0
github.com/xdg-go/scram v1.1.1
github.com/xdg-go/stringprep v1.0.3
github.com/xhit/go-str2duration/v2 v2.1.0
github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d
github.com/zeebo/xxh3 v1.0.2
gitlab.com/nyarla/go-crypt v0.0.0-20160106005555-d9a5dc2b789b
go.mongodb.org/mongo-driver v1.7.5
go.opencensus.io v0.24.0
go.opentelemetry.io/auto/sdk v1.2.1
go.opentelemetry.io/contrib/detectors/gcp v1.42.0
go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0
go.opentelemetry.io/otel v1.44.0
go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0
go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0
go.opentelemetry.io/otel/exporters/zipkin v1.44.0
go.opentelemetry.io/otel/metric v1.44.0
go.opentelemetry.io/otel/sdk v1.44.0
go.opentelemetry.io/otel/sdk/metric v1.44.0
go.opentelemetry.io/otel/trace v1.44.0
go.opentelemetry.io/proto/otlp v1.10.0
go.uber.org/atomic v1.7.0
go.uber.org/goleak v1.3.0
go.yaml.in/yaml/v2 v2.4.4
go.yaml.in/yaml/v3 v3.0.4
golang.org/x/crypto v0.54.0
golang.org/x/exp v0.0.0-20230315142452-642cacee5cc0
golang.org/x/mod v0.37.0
golang.org/x/net v0.57.0
golang.org/x/oauth2 v0.36.0
golang.org/x/sync v0.22.0
golang.org/x/sys v0.47.0
golang.org/x/term v0.45.0
golang.org/x/text v0.40.0
golang.org/x/time v0.15.0
golang.org/x/tools v0.47.0
golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028
gonum.org/v1/gonum v0.17.0
google.golang.org/api v0.169.0
google.golang.org/appengine v1.6.8
google.golang.org/genproto v0.0.0-20240213162025-012b6fc9bca9
google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa
google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa
google.golang.org/grpc v1.81.1
google.golang.org/protobuf v1.36.11
gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
gopkg.in/inf.v0 v0.9.1
gopkg.in/yaml.v3 v3.0.1
lukechampine.com/uint128 v1.2.0
modernc.org/b v1.0.0
modernc.org/cc/v3 v3.36.3
modernc.org/ccgo/v3 v3.16.9
modernc.org/db v1.0.0
modernc.org/file v1.0.0
modernc.org/fileutil v1.0.0
modernc.org/golex v1.0.0
modernc.org/internal v1.0.0
modernc.org/libc v1.17.1
modernc.org/lldb v1.0.0
modernc.org/mathutil v1.5.0
modernc.org/memory v1.2.1
modernc.org/opt v0.1.3
modernc.org/ql v1.0.0
modernc.org/sortutil v1.1.0
modernc.org/sqlite v1.18.1
modernc.org/strutil v1.1.3
modernc.org/token v1.0.0
modernc.org/zappy v1.0.0\n```

## Vulnerability Scan (govulncheck)

_govulncheck is not installed._\n\nInstall with: `go install golang.org/x/vuln/cmd/govulncheck@latest`\n\nWould check for known CVEs in Go dependencies.\n

## Outdated Dependencies

The following dependencies have newer versions available:
| Dependency | Current | Available |
|---|---|---|
| cel.dev/expr | v0.25.1 | v0.25.2 |
| cloud.google.com/go | v0.112.1 | v0.123.0 |
| cloud.google.com/go/compute | v1.25.1 | v1.65.0 |
| cloud.google.com/go/iam | v1.1.6 | v1.12.0 |
| cloud.google.com/go/longrunning | v0.5.5 | v1.2.0 |
| cloud.google.com/go/spanner | v1.56.0 | v1.94.0 |
| cloud.google.com/go/storage | v1.38.0 | v1.64.0 |
| github.com/99designs/keyring | v1.2.1 | v1.2.2 |
| github.com/Azure/azure-sdk-for-go/sdk/azcore | v1.4.0 | v1.22.0 |
| github.com/Azure/azure-sdk-for-go/sdk/internal | v1.1.2 | v1.12.0 |
| github.com/Azure/azure-sdk-for-go/sdk/storage/azblob | v1.0.0 | v1.8.0 |
| github.com/Azure/go-ansiterm | v0.0.0-20230124172434-306776ec8161 | v0.0.0-20250102033503-faa5f7b0171c |
| github.com/Azure/go-autorest/autorest/adal | v0.9.16 | (retracted) [v0.9.24] (deprecated) |
| github.com/Azure/go-autorest/autorest/date | v0.3.0 | v0.3.1] (deprecated) |
| github.com/Azure/go-autorest/logger | v0.2.1 | v0.2.2] (deprecated) |
| github.com/Azure/go-autorest/tracing | v0.6.0 | v0.6.1] (deprecated) |
| github.com/ClickHouse/clickhouse-go | v1.4.3 | v1.5.4 |
| github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp | v1.31.0 | v1.35.0 |
| github.com/IBM/sarama | v1.43.1 | v1.60.1 |
| github.com/andybalholm/brotli | v1.0.4 | v1.2.2 |
| github.com/apache/thrift | v0.16.0 | v0.24.0 |
| github.com/aws/aws-sdk-go | v1.49.6 | v1.55.8] (deprecated) |
| github.com/aws/aws-sdk-go-v2 | v1.16.16 | v1.43.2 |
| github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream | v1.4.8 | v1.7.15 |
| github.com/aws/aws-sdk-go-v2/credentials | v1.12.20 | v1.19.32 |
| github.com/aws/aws-sdk-go-v2/feature/s3/manager | v1.11.33 | v1.22.37] (deprecated) |
| github.com/aws/aws-sdk-go-v2/internal/configsources | v1.1.23 | v1.4.33 |
| github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 | v2.4.17 | v2.7.33 |
| github.com/aws/aws-sdk-go-v2/internal/v4a | v1.0.14 | v1.4.34 |
| github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding | v1.9.9 | v1.13.14 |
| github.com/aws/aws-sdk-go-v2/service/internal/checksum | v1.1.18 | v1.9.26 |
| github.com/aws/aws-sdk-go-v2/service/internal/presigned-url | v1.9.17 | v1.13.33 |
| github.com/aws/aws-sdk-go-v2/service/internal/s3shared | v1.13.17 | v1.19.34 |
| github.com/aws/aws-sdk-go-v2/service/s3 | v1.27.11 | v1.106.2 |
| github.com/aws/smithy-go | v1.13.3 | v1.27.5 |
| github.com/cenkalti/backoff/v4 | v4.1.2 | v4.3.0 |
| github.com/cloudflare/golz4 | v0.0.0-20150217214814-ef862a3cdc58 | v0.0.0-20240916140612-caecf3c00c06 |
| github.com/cockroachdb/cockroach-go/v2 | v2.1.1 | v2.4.3 |
| github.com/cznic/mathutil | v0.0.0-20180504122225-ca4c9f2c1369 | v0.0.0-20181122101859-297441e03548 |
| github.com/danieljoos/wincred | v1.1.2 | v1.2.3 |
| github.com/dhui/dktest | v0.4.3 | v0.4.6 |
| github.com/docker/docker | v27.2.0+incompatible | v28.5.2+incompatible |
| github.com/docker/go-connections | v0.5.0 | v0.8.1 |
| github.com/dvsekhvalnov/jose2go | v1.6.0 | v1.8.0 |
| github.com/eapache/go-resiliency | v1.6.0 | v1.7.0 |
| github.com/edsrzf/mmap-go | v0.0.0-20170320065105-0bce6a688712 | v1.2.0 |
| github.com/felixge/httpsnoop | v1.0.4 | v1.1.0 |
| github.com/fsouza/fake-gcs-server | v1.17.0 | v1.55.1 |
| github.com/gabriel-vasile/mimetype | v1.4.1 | v1.4.15 |
| github.com/go-chi/chi/v5 | v5.1.0 | v5.3.1 |
| github.com/go-logr/logr | v1.4.3 | v1.4.4 |
| github.com/go-sql-driver/mysql | v1.5.0 | v1.10.0 |
| github.com/go-stack/stack | v1.8.0 | v1.8.1 |
| github.com/gobuffalo/here | v0.6.0 | v0.6.7 |
| github.com/goccy/go-json | v0.9.11 | v0.10.6 |
| github.com/gocql/gocql | v0.0.0-20210515062232-b7ef815b4556 | v1.7.0 |
| github.com/golang-jwt/jwt/v4 | v4.4.2 | v4.5.2 |
| github.com/golang-migrate/migrate/v4 | v4.18.1 | v4.19.1 |
| github.com/golang-sql/civil | v0.0.0-20190719163853-cb61b32ac6fe | v0.0.0-20220223132316-b832511892a9 |
| github.com/golang/groupcache | v0.0.0-20210331224755-41bb18bfe9da | v0.0.0-20241129210726-2c02b8208cf8 |
| github.com/golang/snappy | v0.0.4 | v1.0.0 |
| github.com/google/flatbuffers | v2.0.8+incompatible | v25.12.19+incompatible |
| github.com/google/go-querystring | v1.1.0 | v1.2.0 |
| github.com/google/pprof | v0.0.0-20210407192527-94a9f03dee38 | v0.0.0-20260709232956-b9395ee17fa0 |
| github.com/google/s2a-go | v0.1.7 | v0.1.9 |
| github.com/googleapis/enterprise-certificate-proxy | v0.3.2 | v0.3.19 |
| github.com/googleapis/gax-go/v2 | v2.12.2 | v2.23.0 |
| github.com/gorilla/handlers | v1.4.2 | v1.5.2 |
| github.com/gorilla/mux | v1.7.4 | v1.8.1 |
| github.com/jackc/pgerrcode | v0.0.0-20220416144525-469b46aa5efa | v0.0.0-20250907135507-afb5586c32a6 |
| github.com/jackc/pgtype | v1.14.0 | v1.14.4 |
| github.com/jackc/pgx/v4 | v4.18.2 | v4.18.3 |
| github.com/jackc/pgx/v5 | v5.7.1 | v5.10.0 |
| github.com/k0kubun/pp | v2.3.0+incompatible | v3.0.1+incompatible |
| github.com/klauspost/cpuid/v2 | v2.0.9 | v2.4.0 |
| github.com/ktrysmt/go-bitbucket | v0.6.4 | v0.10.0 |
| github.com/lib/pq | v1.10.9 | v1.12.3 |
| github.com/markbates/pkger | v0.15.1 | v0.17.1 |
| github.com/mattn/go-colorable | v0.1.6 | v0.1.15 |
| github.com/mattn/go-isatty | v0.0.16 | v0.0.24 |
| github.com/mattn/go-sqlite3 | v1.14.22 | v1.14.49 |
| github.com/microsoft/go-mssqldb | v1.0.0 | v1.10.0 |
| github.com/mitchellh/mapstructure | v1.1.2 | v1.5.0 |
| github.com/moby/term | v0.5.0 | v0.5.2 |
| github.com/morikuni/aec | v1.0.0 | v1.1.0 |
| github.com/mutecomm/go-sqlcipher/v4 | v4.4.0 | v4.4.2 |
| github.com/nakagami/firebirdsql | v0.0.0-20190310045651-3c02a58cfed8 | v0.9.19 |
| github.com/nats-io/nats.go | v1.37.0 | v1.52.0 |
| github.com/nats-io/nkeys | v0.4.7 | v0.4.16 |
| github.com/neo4j/neo4j-go-driver | v1.8.1-0.20200803113522-b626aa943eba | v1.8.3 |
| github.com/onsi/ginkgo | v1.16.4 | v1.16.5 |
| github.com/onsi/ginkgo/v2 | v2.11.0 | v2.32.0 |
| github.com/onsi/gomega | v1.27.10 | v1.42.1 |
| github.com/opencontainers/image-spec | v1.1.0 | v1.1.1 |
| github.com/pierrec/lz4/v4 | v4.1.21 | v4.1.27 |
| github.com/pkg/browser | v0.0.0-20210911075715-681adbf594b8 | v0.0.0-20240102092130-5ac0b6a4141c |
| github.com/rabbitmq/amqp091-go | v1.9.0 | v1.13.0 |
| github.com/rcrowley/go-metrics | v0.0.0-20201227073835-cf1acfcdf475 | v0.0.0-20250401214520-65e299d6c5c9 |
| github.com/redis/go-redis/v9 | v9.7.0 | v9.21.0 |
| github.com/remyoudompheng/bigfft | v0.0.0-20200410134404-eec4a21b6bb0 | v0.0.0-20230129092748-24d4a6f8daec |
| github.com/rogpeppe/go-internal | v1.14.1 | v1.15.0 |
| github.com/rqlite/gorqlite | v0.0.0-20230708021416-2acd02b70b79 | v0.0.0-20260504155303-50d445fd0ab9 |
| github.com/shopspring/decimal | v1.2.0 | v1.4.0 |
| github.com/sirupsen/logrus | v1.9.3 | v1.9.4 |
| github.com/snowflakedb/gosnowflake | v1.6.19 | v1.19.1 |
| github.com/spiffe/go-spiffe/v2 | v2.6.0 | v2.8.1 |
| github.com/stretchr/objx | v0.5.2 | v0.5.3 |
| github.com/xanzy/go-gitlab | v0.15.0 | v0.115.0] (deprecated) |
| github.com/xdg-go/scram | v1.1.1 | v1.2.0 |
| github.com/xdg-go/stringprep | v1.0.3 | v1.0.4 |
| github.com/youmark/pkcs8 | v0.0.0-20181117223130-1be2e3e5546d | v0.0.0-20240726163527-a2c0da244d78 |
| github.com/zeebo/xxh3 | v1.0.2 | v1.1.0 |
| go.mongodb.org/mongo-driver | v1.7.5 | v1.17.9] (deprecated) |
| go.opentelemetry.io/contrib/detectors/gcp | v1.42.0 | v1.44.0 |
| go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc | v0.49.0 | v0.69.0 |
| go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp | v0.54.0 | v0.69.0 |
| go.opentelemetry.io/proto/otlp | v1.10.0 | v1.11.0 |
| go.uber.org/atomic | v1.7.0 | v1.11.0 |
| go.yaml.in/yaml/v3 | v3.0.4 | v3.0.5 |
| golang.org/x/exp | v0.0.0-20230315142452-642cacee5cc0 | v0.0.0-20260727155853-b88d891fe743 |
| golang.org/x/mod | v0.37.0 | v0.38.0 |
| golang.org/x/tools | v0.47.0 | v0.48.0 |
| golang.org/x/xerrors | v0.0.0-20231012003039-104605ab7028 | v0.0.0-20240903120638-7835f813f4da |
| google.golang.org/api | v0.169.0 | v0.291.0 |
| google.golang.org/genproto | v0.0.0-20240213162025-012b6fc9bca9 | v0.0.0-20260729162451-8efbd57d26e0 |
| google.golang.org/genproto/googleapis/api | v0.0.0-20260526163538-3dc84a4a5aaa | v0.0.0-20260729162451-8efbd57d26e0 |
| google.golang.org/genproto/googleapis/rpc | v0.0.0-20260526163538-3dc84a4a5aaa | v0.0.0-20260729162451-8efbd57d26e0 |
| google.golang.org/grpc | v1.81.1 | v1.82.1 |
| lukechampine.com/uint128 | v1.2.0 | v1.3.0 |
| modernc.org/b | v1.0.0 | v1.1.0 |
| modernc.org/cc/v3 | v3.36.3 | v3.41.0 |
| modernc.org/ccgo/v3 | v3.16.9 | v3.17.0 |
| modernc.org/db | v1.0.0 | v1.3.1 |
| modernc.org/file | v1.0.0 | v1.1.3 |
| modernc.org/fileutil | v1.0.0 | v1.4.0 |
| modernc.org/golex | v1.0.0 | v1.1.0 |
| modernc.org/internal | v1.0.0 | v1.1.12 |
| modernc.org/libc | v1.17.1 | v1.74.4 |
| modernc.org/lldb | v1.0.0 | v1.0.8 |
| modernc.org/mathutil | v1.5.0 | v1.7.1 |
| modernc.org/memory | v1.2.1 | v1.11.0 |
| modernc.org/opt | v0.1.3 | v0.2.0 |
| modernc.org/ql | v1.0.0 | v1.5.2 |
| modernc.org/sortutil | v1.1.0 | v1.2.1 |
| modernc.org/sqlite | v1.18.1 | v1.55.0 |
| modernc.org/strutil | v1.1.3 | v1.2.1 |
| modernc.org/token | v1.0.0 | v1.1.0 |
| modernc.org/zappy | v1.0.0 | v1.1.0 |


## Module: github.com/renos-project/renos/renctl

**Path:** `/home/ren/Projects/RenOS/renctl`\n

## Direct Dependencies

These are the direct (non-indirect) dependencies listed in `go.mod`:
| Dependency | Version |
|---|---|

## go mod verify

```\nall modules verified\n```

## All Dependencies (go list -m all)

```\ngithub.com/renos-project/renos/renctl\n```

## Vulnerability Scan (govulncheck)

_govulncheck is not installed._\n\nInstall with: `go install golang.org/x/vuln/cmd/govulncheck@latest`\n\nWould check for known CVEs in Go dependencies.\n

## Outdated Dependencies

All dependencies are up to date.


## Module: github.com/renos-project/renos/sdk/go

**Path:** `/home/ren/Projects/RenOS/sdk/go`\n

## Direct Dependencies

These are the direct (non-indirect) dependencies listed in `go.mod`:
| Dependency | Version |
|---|---|

## go mod verify

```\nall modules verified\n```

## All Dependencies (go list -m all)

```\ngithub.com/renos-project/renos/sdk/go
github.com/gorilla/websocket v1.5.3\n```

## Vulnerability Scan (govulncheck)

_govulncheck is not installed._\n\nInstall with: `go install golang.org/x/vuln/cmd/govulncheck@latest`\n\nWould check for known CVEs in Go dependencies.\n

## Outdated Dependencies

All dependencies are up to date.


## Module: github.com/renos-project/renos/installer/upgrade

**Path:** `/home/ren/Projects/RenOS/installer/upgrade`\n

## Direct Dependencies

These are the direct (non-indirect) dependencies listed in `go.mod`:
| Dependency | Version |
|---|---|

## go mod verify

```\nall modules verified\n```

## All Dependencies (go list -m all)

```\ngithub.com/renos-project/renos/installer/upgrade\n```

## Vulnerability Scan (govulncheck)

_govulncheck is not installed._\n\nInstall with: `go install golang.org/x/vuln/cmd/govulncheck@latest`\n\nWould check for known CVEs in Go dependencies.\n

## Outdated Dependencies

All dependencies are up to date.


## Module: github.com/renos-project/renos/installer/wizard

**Path:** `/home/ren/Projects/RenOS/installer/wizard`\n

## Direct Dependencies

These are the direct (non-indirect) dependencies listed in `go.mod`:
| Dependency | Version |
|---|---|

## go mod verify

```\nall modules verified\n```

## All Dependencies (go list -m all)

```\ngithub.com/renos-project/renos/installer/wizard\n```

## Vulnerability Scan (govulncheck)

_govulncheck is not installed._\n\nInstall with: `go install golang.org/x/vuln/cmd/govulncheck@latest`\n\nWould check for known CVEs in Go dependencies.\n

## Outdated Dependencies

All dependencies are up to date.


## Module: github.com/renos-project/renos/core/cmd/renos-plugin

**Path:** `/home/ren/Projects/RenOS/core/cmd/renos-plugin`\n

## Direct Dependencies

These are the direct (non-indirect) dependencies listed in `go.mod`:
| Dependency | Version |
|---|---|

## go mod verify

```\nall modules verified\n```

## All Dependencies (go list -m all)

```\ngithub.com/renos-project/renos/core/cmd/renos-plugin\n```

## Vulnerability Scan (govulncheck)

_govulncheck is not installed._\n\nInstall with: `go install golang.org/x/vuln/cmd/govulncheck@latest`\n\nWould check for known CVEs in Go dependencies.\n

## Outdated Dependencies

All dependencies are up to date.


## npm Dependencies


## Package: renos-hub

**Path:** `/home/ren/Projects/RenOS/hub`\n
| Dependency | Version | Type |
|---|---|---|
| @tanstack/react-query | ^5.59.0 | production |
| @xterm/addon-clipboard | ^0.2.0 | production |
| @xterm/addon-fit | ^0.11.0 | production |
| @xterm/addon-search | ^0.16.0 | production |
| @xterm/addon-web-links | ^0.12.0 | production |
| @xterm/xterm | ^6.0.0 | production |
| clsx | ^2.1.1 | production |
| lucide-react | ^0.451.0 | production |
| next | ^15.0.0 | production |
| react | ^19.0.0 | production |
| react-dom | ^19.0.0 | production |
| tailwind-merge | ^2.5.0 | production |
| xterm | ^5.3.0 | production |
| zustand | ^5.0.0 | production |
| @testing-library/jest-dom | ^6.6.0 | development |
| @testing-library/react | ^16.0.0 | development |
| @types/node | ^22.0.0 | development |
| @types/react | ^19.0.0 | development |
| @types/react-dom | ^19.0.0 | development |
| autoprefixer | ^10.4.0 | development |
| eslint | ^9.0.0 | development |
| eslint-config-next | ^15.0.0 | development |
| jsdom | ^25.0.0 | development |
| postcss | ^8.4.0 | development |
| prettier | ^3.3.0 | development |
| tailwindcss | ^3.4.0 | development |
| typescript | ^5.6.0 | development |
| vitest | ^2.1.0 | development |
