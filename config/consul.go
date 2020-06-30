package config

import (
	"errors"
	"github.com/hashicorp/consul/api"
)

type ConsulConfig struct {
	param *ConsulParam
	cfg   *api.Config
}

type ConsulParam struct {
	KVPath string
}

// S3 implements a s3 provider.
type ConsulClient struct {
	*api.Client
	*ConsulParam
}

// Provider returns a provider that takes a simples3 config.
func ConsulProvider(cfg *ConsulConfig) *ConsulClient {

	client, err := api.NewClient(cfg.cfg)
	if err != nil {
		panic(err)
	}

	return &ConsulClient{client, cfg.param}

}

// ReadBytes reads the contents of a file on s3 and returns the bytes.
func (r *ConsulClient) ReadBytes() ([]byte, error) {

	kv := r.KV()

	pair, _, err := kv.Get(r.ConsulParam.KVPath, nil)
	if err != nil {
		return nil, err
	}

	data := pair.Value
	return data, err

}

// Read returns the raw bytes for parsing.
func (r *ConsulClient) Read() (map[string]interface{}, error) {
	return nil, errors.New("buf provider does not support this method")
}

// Watch is not supported.
func (r *ConsulClient) Watch(cb func(event interface{}, err error)) error {
	return errors.New("S3 provider does not support this method")
}
