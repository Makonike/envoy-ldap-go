package envoy_ldap_go

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
	"google.golang.org/protobuf/types/known/anypb"

	xds "github.com/cncf/xds/go/xds/type/v3"
)

func init() {
	http.RegisterHttpFilterConfigFactory("ldap-auth", configFactory)
	http.RegisterHttpFilterConfigParser(&parser{})
}

type config struct {
	baseDN   string
	host     string
	port     uint64
	username string
	password string
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
	if baseDN, ok := v.AsMap()["baseDN"].(string); ok {
		conf.baseDN = baseDN
	}
	if host, ok := v.AsMap()["host"].(string); ok {
		conf.host = host
	}
	if port, ok := v.AsMap()["port"].(float64); ok {
		conf.port = uint64(port)
	}
	if username, ok := v.AsMap()["username"].(string); ok {
		conf.username = username
	}
	if password, ok := v.AsMap()["password"].(string); ok {
		conf.password = password
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
