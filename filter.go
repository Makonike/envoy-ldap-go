package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/api"
	"github.com/go-ldap/ldap/v3"
	"net"
	"strings"
	"time"
)

type filter struct {
	callbacks api.FilterCallbackHandler
	config    *config
}

func parseUsernameAndPassword(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", "", false
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}
	cs := string(c)
	username, password, ok = strings.Cut(cs, ":")
	if !ok {
		return "", "", false
	}
	return username, password, true
}

func dial(config *config) (*ldap.Conn, error) {
	return ldap.DialURL(
		// TODO: support TLS
		fmt.Sprintf("ldap://%s:%d", config.host, config.port),
		ldap.DialWithDialer(&net.Dialer{
			Timeout: time.Duration(config.timeout),
		}),
	)
}

// newLdapClient creates a new ldap client.
func newLdapClient(config *config) (*ldap.Conn, error) {
	client, err := dial(config)
	if err != nil {
		fmt.Println("ldap dial error: ", err)
		return nil, err
	}

	err = client.Bind(config.bindDN, config.password)
	// First bind with a read only user
	if err != nil {
		fmt.Println("bind with read only user error: ", err)
		return nil, err
	}
	return client, err
}

// authLdap authenticates the user against the ldap server.
func authLdap(config *config, username, password string) bool {
	if config.filter != "" {
		fmt.Printf("search mode, username: %v\n", username)
		return searchMode(config, username, password)
	}

	// run with bind mode
	fmt.Printf("bind mode, username: %v\n", username)
	client, err := dial(config)
	if err != nil {
		fmt.Println("ldap dial error: ", err)
		return false
	}

	_, err = client.SimpleBind(&ldap.SimpleBindRequest{
		Username: fmt.Sprintf(config.attribute+"=%s,%s", username, config.baseDN),
		Password: password,
	})
	return err == nil
}

func searchMode(config *config, username, password string) (auth bool) {
	client, err := newLdapClient(config)
	if err != nil {
		fmt.Println("ldap dial error: ", err)
		return
	}
	defer func() {
		if client != nil {
			client.Close()
		}
		err := recover()
		if err != nil {
			auth = false
			return
		}
	}()

	req := ldap.NewSearchRequest(config.baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(config.filter, username),
		[]string{config.attribute}, nil)

	sr, err := client.Search(req)
	if err != nil {
		fmt.Println("ldap search error: ", err)
		return
	}

	if len(sr.Entries) != 1 {
		fmt.Println("ldap search not found: ", err)
		return
	}

	userDn := sr.Entries[0].DN
	err = client.Bind(userDn, password)
	if err != nil {
		fmt.Println("ldap bind error: ", err)
		return
	}

	_, err = json.Marshal(sr.Entries[0])
	if err != nil {
		fmt.Println("ldap marshal error: ", err)
		return
	}
	auth = true
	return
}

func (f *filter) verify(header api.RequestHeaderMap) (bool, string) {
	auth, ok := header.Get("authorization")
	if !ok {
		return false, "no Authorization"
	}
	if f.config.cacheTTL > 0 {
		if _, err := f.config.cache.Get(auth); err == nil {
			fmt.Printf("cache hit, auth: %v\n", auth)
			return true, ""
		}
	}

	username, password, ok := parseUsernameAndPassword(auth)
	if !ok {
		return false, "invalid Authorization format"
	}
	ok = authLdap(f.config, username, password)
	if !ok {
		return false, "invalid username or password"
	}
	if f.config.cacheTTL > 0 {
		fmt.Printf("cache set, auth: %v\n", auth)
		_ = f.config.cache.Set(auth, []byte{})
	}
	return true, ""
}

func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	go func() {
		if ok, msg := f.verify(header); !ok {
			// TODO: set the WWW-Authenticate response header
			f.callbacks.SendLocalReply(401, msg, map[string]string{}, 0, "bad-request")
			return
		}
		f.callbacks.Continue(api.Continue)
	}()
	return api.Running
}

func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	return api.Continue
}

func (f *filter) DecodeTrailers(trailers api.RequestTrailerMap) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeTrailers(trailers api.ResponseTrailerMap) api.StatusType {
	return api.Continue
}

func (f *filter) OnDestroy(reason api.DestroyReason) {
}

func main() {
}
