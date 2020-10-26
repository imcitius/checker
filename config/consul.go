package config

import (
	"errors"
	"fmt"
	"github.com/hashicorp/consul/api"
)

type ConsulConfig struct {
	param *ConsulParam
	cfg   *api.Config
}

type ConsulParam struct {
	KVPath string
}

type ConsulClient struct {
	*api.Client
	*ConsulParam
}

// Service represent a Consul service.
type ConsulService struct {
	Name      string
	Tags      []string
	Nodes     []string
	Addresses []string
	Ports     []int
}

var (
	ProjectsCatalog = make(map[string]Project)
)

func ConsulProvider(cfg *ConsulConfig) *ConsulClient {

	client, err := api.NewClient(cfg.cfg)
	if err != nil {
		panic(err)
	}

	return &ConsulClient{client, cfg.param}

}

func (r *ConsulClient) ReadBytes() ([]byte, error) {

	kv := r.KV()

	pair, _, err := kv.Get(r.ConsulParam.KVPath, nil)
	if err != nil {
		return nil, err
	}

	if pair == nil {
		return []byte{}, fmt.Errorf("Cannot get data from consul, empty key")
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
	return errors.New("consul provider does not support this method")
}
