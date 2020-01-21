module github.com/future-architect/docradle

go 1.13

require (
	cuelang.org/go v0.0.15
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/deltam/go-lsd-parametrized v1.4.0
	github.com/future-architect/fluentdpub v0.0.4
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gookit/color v1.2.1
	github.com/joho/godotenv v1.3.0
	github.com/mattn/go-isatty v0.0.11
	github.com/rs/xid v1.2.1
	github.com/rs/zerolog v1.17.2
	github.com/shibukawa/cdiff v0.1.3
	github.com/shirou/gopsutil v2.19.12+incompatible
	github.com/stretchr/testify v1.4.0
	go.pyspa.org/brbundle v1.1.3
	gocloud.dev v0.18.0
	gocloud.dev/pubsub/kafkapubsub v0.18.0
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
)

replace github.com/future-architect/docradle => ./
