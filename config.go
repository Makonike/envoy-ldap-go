package main

import (
	"fmt"
	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	http.RegisterHttpFilterConfigFactory("ldap-auth", configFactory)
	http.RegisterHttpFilterConfigParser(&parser{})
}

type config struct {
	baseDN               string
	host                 string
	port                 uint64
	bindDN               string
	password             string
	attribute            string
	certificateAuthority string
	filter               string
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
	if baseDN, ok := v.AsMap()["base_dn"].(string); ok {
		conf.baseDN = baseDN
	}
	if host, ok := v.AsMap()["host"].(string); ok {
		conf.host = host
	}
	if port, ok := v.AsMap()["port"].(float64); ok {
		conf.port = uint64(port)
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
	if certificateAuthority, ok := v.AsMap()["certificateAuthority"].(string); ok {
		conf.certificateAuthority = certificateAuthority
	}
	if cFilter, ok := v.AsMap()["filter"].(string); ok {
		conf.filter = cFilter
	}
	fmt.Println(conf)
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
