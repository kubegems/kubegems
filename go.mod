module kubegems.io/kubegems

go 1.21

require (
	code.gitea.io/sdk/gitea v0.15.1
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/VividCortex/mysqlerr v1.0.0
	github.com/Xuanwo/go-locale v1.1.0
	github.com/alicebob/miniredis/v2 v2.21.0
	github.com/argoproj/argo-cd/v2 v2.3.4
	github.com/argoproj/argo-rollouts v1.2.2
	github.com/aws/aws-sdk-go-v2 v1.21.0
	github.com/aws/aws-sdk-go-v2/config v1.18.38
	github.com/aws/aws-sdk-go-v2/credentials v1.13.36
	github.com/aws/aws-sdk-go-v2/service/s3 v1.38.5
	github.com/banzaicloud/logging-operator/pkg/sdk v0.7.26
	github.com/casbin/casbin/v2 v2.73.0
	github.com/casbin/gorm-adapter/v3 v3.16.1
	github.com/containerd/containerd v1.7.0
	github.com/containers/image/v5 v5.27.1-0.20230908144338-a9b09b31f323
	github.com/coreos/go-oidc/v3 v3.6.0
	github.com/creack/pty v1.1.18
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21
	github.com/emersion/go-smtp v0.15.0
	github.com/emicklei/go-restful-openapi/v2 v2.9.0
	github.com/emicklei/go-restful/v3 v3.10.1
	github.com/evanphx/json-patch/v5 v5.6.0
	github.com/gin-gonic/gin v1.9.0
	github.com/go-git/go-billy/v5 v5.3.1
	github.com/go-git/go-git/v5 v5.4.2
	github.com/go-ldap/ldap/v3 v3.2.4
	github.com/go-logr/logr v1.2.4
	github.com/go-logr/zapr v1.2.4
	github.com/go-oauth2/oauth2/v4 v4.5.1
	github.com/go-openapi/spec v0.20.9
	github.com/go-playground/locales v0.14.1
	github.com/go-playground/universal-translator v0.18.1
	github.com/go-playground/validator/v10 v10.12.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-redsync/redsync/v4 v4.5.0
	github.com/go-resty/resty/v2 v2.7.0
	github.com/go-sql-driver/mysql v1.7.0
	github.com/gogo/protobuf v1.3.2
	github.com/goharbor/harbor/src v0.0.0-20210616083956-c39345da96d8
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/go-cmp v0.5.9
	github.com/google/uuid v1.3.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/go-envparse v0.1.0
	github.com/hashicorp/go-version v1.5.0
	github.com/kiali/kiali v1.43.0
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0
	github.com/mattbaird/jsonpatch v0.0.0-20200820163806-098863c1fc24
	github.com/oam-dev/kubevela v1.1.8
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/opencontainers/distribution-spec v1.0.1
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator v0.46.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.46.0
	github.com/prometheus/alertmanager v0.23.0
	github.com/prometheus/client_golang v1.14.0
	github.com/prometheus/common v0.37.0
	github.com/prometheus/prometheus v2.5.0+incompatible
	github.com/robfig/cron/v3 v3.0.1
	github.com/seldonio/seldon-core/operator v1.14.0
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.16.0
	github.com/stoewer/go-strcase v1.2.0
	github.com/stretchr/testify v1.8.4
	github.com/swaggo/files v0.0.0-20210815190702-a29dd2bc99b2
	github.com/swaggo/gin-swagger v1.3.1
	github.com/swaggo/swag v1.8.1
	github.com/zitadel/oidc v1.7.0
	go.mongodb.org/mongo-driver v1.12.1
	go.opentelemetry.io/contrib/instrumentation/runtime v0.43.0
	go.opentelemetry.io/otel v1.17.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v0.40.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.17.0
	go.opentelemetry.io/otel/metric v1.17.0
	go.opentelemetry.io/otel/sdk v1.17.0
	go.opentelemetry.io/otel/sdk/metric v0.40.0
	go.opentelemetry.io/otel/trace v1.17.0
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.16.0
	golang.org/x/exp v0.0.0-20231226003508-02704c960a9b
	golang.org/x/mod v0.14.0
	golang.org/x/net v0.19.0
	golang.org/x/oauth2 v0.12.0
	golang.org/x/sync v0.5.0
	golang.org/x/sys v0.15.0
	golang.org/x/text v0.14.0
	golang.org/x/time v0.1.0
	google.golang.org/grpc v1.58.0
	google.golang.org/protobuf v1.31.0
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/datatypes v1.0.2
	gorm.io/driver/mysql v1.5.1
	gorm.io/gorm v1.25.1
	helm.sh/helm/v3 v3.8.2
	istio.io/api v0.0.0-20220512212136-561ffec82582
	istio.io/client-go v1.13.4-0.20220512212836-a717f1acbd1f
	istio.io/istio v0.0.0-20220516185659-202e88863858
	k8s.io/api v0.26.2
	k8s.io/apiextensions-apiserver v0.23.6
	k8s.io/apimachinery v0.28.1
	k8s.io/cli-runtime v0.23.6
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubectl v0.23.5
	k8s.io/kubernetes v1.23.1
	k8s.io/metrics v0.23.6
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2
	kubegems.io/configer v1.1.6
	kubegems.io/ingress-nginx-operator v0.2.1
	kubegems.io/library v0.0.0-20230911041405-0b02d2dbc2b4
	kubegems.io/modelx v0.1.7
	sigs.k8s.io/controller-runtime v0.11.2
	sigs.k8s.io/kustomize/api v0.11.4
	sigs.k8s.io/kustomize/kyaml v0.13.6
	sigs.k8s.io/yaml v1.3.0
)

require (
	cloud.google.com/go v0.110.6 // indirect
	cloud.google.com/go/compute v1.23.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/logging v1.7.0 // indirect
	cloud.google.com/go/longrunning v0.5.1 // indirect
	emperror.dev/errors v0.8.0 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230106234847-43070de90fa1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.27 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.18 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20200615164410-66371956d46c // indirect
	github.com/BurntSushi/toml v1.3.2 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/Masterminds/squirrel v1.5.2 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/argoproj/gitops-engine v0.6.2 // indirect
	github.com/argoproj/pkg v0.11.1-0.20211203175135-36c59d8fafe0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/astaxie/beego v1.12.1 // indirect
	github.com/aws/aws-sdk-go v1.44.5 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.41 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.35 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.42 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.1.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.15.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.13.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.15.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.21.5 // indirect
	github.com/aws/smithy-go v1.14.2 // indirect
	github.com/banzaicloud/operator-tools v0.28.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bombsimon/logrusr/v2 v2.0.1 // indirect
	github.com/bradleyfalzon/ghinstallation/v2 v2.0.4 // indirect
	github.com/bytedance/sonic v1.8.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/containers/libtrust v0.0.0-20230121012942-c1716e8a8d01 // indirect
	github.com/containers/ocicrypt v1.1.8 // indirect
	github.com/containers/storage v1.49.0 // indirect
	github.com/coreos/go-oidc v2.2.1+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/cli v20.10.16+incompatible // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker v24.0.6+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.0 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/fvbommel/sortorder v1.0.1 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/glebarez/go-sqlite v1.20.3 // indirect
	github.com/glebarez/sqlite v1.7.0 // indirect
	github.com/go-asn1-ber/asn1-ber v1.5.1 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.0 // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-kit/log v0.2.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-redis/cache/v8 v8.4.2 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.0 // indirect
	github.com/gocraft/work v0.5.1 // indirect
	github.com/golang-jwt/jwt/v4 v4.2.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.11.0 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-github/v41 v41.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/s2a-go v0.1.4 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.11.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/schema v1.2.0 // indirect
	github.com/gorilla/securecookie v1.1.1 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.4 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/iancoleman/orderedmap v0.2.0 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.13.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.1 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.12.0 // indirect
	github.com/jackc/pgx/v4 v4.17.2 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jedib0t/go-pretty/v6 v6.4.7 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmoiron/sqlx v1.3.4 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kedacore/keda/v2 v2.7.1 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/leodido/go-urn v1.2.2 // indirect
	github.com/lib/pq v1.10.5 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mholt/archiver/v4 v4.0.0-alpha.7 // indirect
	github.com/microsoft/go-mssqldb v0.17.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/montanaflynn/stats v0.6.6 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/nitishm/engarde v0.1.1 // indirect
	github.com/nwaples/rardecode/v2 v2.0.0-beta.2 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/oam-dev/terraform-controller v0.7.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc4 // indirect
	github.com/opencontainers/runc v1.1.9 // indirect
	github.com/opencontainers/runtime-spec v1.1.0 // indirect
	github.com/opencontainers/selinux v1.11.0 // indirect
	github.com/openshift/api v0.0.0-20211209135129-c58d9f695577 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20180306154005-525d0eb5f91d // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common/sigv4 v0.1.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230126093431-47fa9a501578 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/rs/zerolog v1.21.0 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20210614095031-55d5740dbbcc // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/therootcompany/xz v1.0.1 // indirect
	github.com/tidwall/btree v1.3.1 // indirect
	github.com/tidwall/buntdb v1.2.9 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/grect v0.1.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/rtred v0.1.2 // indirect
	github.com/tidwall/tinyqueue v0.1.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/uber/jaeger-client-go v2.29.1+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.0+incompatible // indirect
	github.com/ugorji/go/codec v1.2.9 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	github.com/vjeantet/grok v1.0.0 // indirect
	github.com/vmihailenco/go-tinylfu v0.2.1 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.4 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	github.com/yuin/gopher-lua v0.0.0-20210529063254-f4c35e4016d9 // indirect
	github.com/zitadel/logging v0.3.4 // indirect
	go.etcd.io/etcd/api/v3 v3.5.9 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.9 // indirect
	go.etcd.io/etcd/client/v3 v3.5.9 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.17.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.starlark.net v0.0.0-20211013185944-b0039bd2cfe3 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/goleak v1.2.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/arch v0.0.0-20210923205945-b76863e36670 // indirect
	golang.org/x/term v0.15.0 // indirect
	golang.org/x/tools v0.16.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/api v0.126.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	gopkg.in/gorp.v1 v1.7.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gorm.io/driver/postgres v1.4.4 // indirect
	gorm.io/driver/sqlserver v1.4.1 // indirect
	gorm.io/plugin/dbresolver v1.3.0 // indirect
	istio.io/gogo-genproto v0.0.0-20220413184606-e76735307e2d // indirect
	istio.io/pkg v0.0.0-20220413180906-765b7512325f // indirect
	k8s.io/apiserver v0.26.2 // indirect
	k8s.io/cloud-provider v0.23.5 // indirect
	k8s.io/component-base v0.26.2 // indirect
	k8s.io/component-helpers v0.23.6 // indirect
	k8s.io/cri-api v0.26.2 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-aggregator v0.23.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220803164354-a70c9af30aea // indirect
	k8s.io/mount-utils v0.23.5 // indirect
	knative.dev/pkg v0.0.0-20220502225657-4fced0164c9a // indirect
	modernc.org/libc v1.22.2 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.20.3 // indirect
	oras.land/oras-go v1.1.1 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)

replace (
	github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.10.2-0.20220119144013-5ce31a86d483
	github.com/nginxinc/nginx-ingress-operator => github.com/kubegems/nginx-ingress-operator v0.3.2
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20210421143221-52df5ef7a3be
	k8s.io/api => k8s.io/api v0.23.5
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.5
	k8s.io/apiserver => k8s.io/apiserver v0.23.5
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.5
	k8s.io/client-go => k8s.io/client-go v0.23.5
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.5
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.5
	k8s.io/code-generator => k8s.io/code-generator v0.23.5
	k8s.io/component-base => k8s.io/component-base v0.23.5
	k8s.io/component-helpers => k8s.io/component-helpers v0.23.5
	k8s.io/controller-manager => k8s.io/controller-manager v0.23.5
	k8s.io/cri-api => k8s.io/cri-api v0.23.5
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.5
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.5
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.5
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.5
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.5
	k8s.io/kubectl => k8s.io/kubectl v0.23.5
	k8s.io/kubelet => k8s.io/kubelet v0.23.5
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.5
	k8s.io/metrics => k8s.io/metrics v0.23.5
	k8s.io/mount-utils => k8s.io/mount-utils v0.23.5
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.23.5
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.5
	github.com/wercker/stern => github.com/oam-dev/stern v1.13.2
)
