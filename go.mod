module github.com/je4/s3image/v2

go 1.16

replace github.com/je4/s3image/v2 => ./

require gopkg.in/gographics/imagick.v3 v3.4.0

require (
	github.com/goph/emperror v0.17.2
	github.com/je4/zsearch/v2 v2.0.0-20211119134546-2e2b76e7d46a
	github.com/minio/minio-go/v7 v7.0.15
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/errors v0.9.1
	gopkg.in/gographics/imagick.v1 v1.1.2
	gopkg.in/gographics/imagick.v2 v2.6.0
)

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/gorilla/mux v1.8.0
	github.com/je4/utils/v2 v2.0.6
)
