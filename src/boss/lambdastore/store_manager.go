package lambdastore

import (
	"fmt"

	"github.com/open-lambda/open-lambda/ol/boss/config"
)

// NewLambdaStore returns a LambdaStore implementation based on the platform in config
func NewLambdaStore(conf *config.Config) (*LambdaStore, error) {
	switch conf.Platform {
	case "local":
		storePath := conf.Local.LambdaStoreLocal
		localStore, err := NewLocalLambdaStore(storePath)
		if err != nil {
			return nil, err
		}
		return &LambdaStore{
			LambdaStorePlatform: localStore,
			Platform:            "local",
		}, nil

	case "gcp":
		bucket := conf.Gcp.LambdaStoreGCS.Bucket
		prefix := conf.Gcp.LambdaStoreGCS.Prefix
		gcsStore, err := NewGCSLambdaStore(bucket, prefix)
		if err != nil {
			return nil, err
		}
		return &LambdaStore{
			LambdaStorePlatform: gcsStore,
			Platform:            "gcp",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported platform for lambda store: %s", conf.Platform)
	}
}
