package main

import (
	"context"
	"github.com/allegro/bigcache/v3"
	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
	"google.golang.org/protobuf/types/known/anypb"
	"time"
)

func init() {
	http.RegisterHttpFilterConfigFactory("ldap-auth", configFactory)
	http.RegisterHttpFilterConfigParser(&parser{})
}

type config struct {
	host      string
	port      uint64
	baseDN    string
	attribute string
	bindDN    string
	password  string
	filter    string
	cacheTTL  int32
	timeout   int32
	cache     *bigcache.BigCache
}

type parser struct {
}

func (p *parser) Parse(any *anypb.Any) (interface{}, error) {
	configStruct := &xds.TypedStruct{}
	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	v := configStruct.Value
	conf := &config{}
	if host, ok := v.AsMap()["host"].(string); ok {
		conf.host = host
	}
	if port, ok := v.AsMap()["port"].(float64); ok {
		conf.port = uint64(port)
	}
	if baseDN, ok := v.AsMap()["base_dn"].(string); ok {
		conf.baseDN = baseDN
	}
	if attribute, ok := v.AsMap()["attribute"].(string); ok {
		conf.attribute = attribute
	}
	if bindDN, ok := v.AsMap()["bind_dn"].(string); ok {
		conf.bindDN = bindDN
	}
	if password, ok := v.AsMap()["bind_password"].(string); ok {
		conf.password = password
	}
	if cFilter, ok := v.AsMap()["filter"].(string); ok {
		conf.filter = cFilter
	}
	if cacheTTL, ok := v.AsMap()["cache_ttl"].(float64); ok {
		conf.cacheTTL = int32(cacheTTL)
	}
	// default is 0, which means no cache
	var err error
	if conf.cacheTTL > 0 {
		conf.cache, err = bigcache.New(context.Background(), bigcache.DefaultConfig(time.Duration(conf.cacheTTL)*time.Second))
		if err != nil {
			panic(err)
		}
	}

	if timeout, ok := v.AsMap()["timeout"].(int32); ok {
		conf.timeout = timeout
	}
	if conf.timeout == 0 {
		conf.timeout = 60000
	}
	return conf, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	panic("TODO")
}

func configFactory(c interface{}) api.StreamFilterFactory {
	conf, ok := c.(*config)
	if !ok {
		panic("unexpected config type")
	}
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &filter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}
