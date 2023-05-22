package main

import (
	"encoding/json"
	"fmt"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/api"
	"github.com/go-ldap/ldap/v3"
	"strings"
)

type filter struct {
	callbacks api.FilterCallbackHandler
	config    *config
}

// todo: encrypt the message
func parseUsernameAndPassword(auth string) (username, password string, ok bool) {
	raw := strings.Split(auth, " ")
	if len(raw) != 2 {
		return "", "", false
	}
	return raw[0], raw[1], true
}

// newLdapClient creates a new ldap client.
func newLdapClient(config *config) (*ldap.Conn, error) {
	client, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", config.host, config.port))
	if err != nil {
		fmt.Println("ldap dial error: ", err)
		return nil, err
	}

	err = client.Bind(config.bindDN, config.password)
	// First bind with a read only user
	if err != nil {
		fmt.Println("new ldap client, ldap bind error: ", err)
		return nil, err
	}
	return client, err
}

// authLdap authenticates the user against the ldap server.
func authLdap(config *config, username, password string) (auth bool, meta string) {
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
		fmt.Println("ldap search error, not found: ", err)
		return
	}

	userdn := sr.Entries[0].DN
	err = client.Bind(userdn, password)
	if err != nil {
		fmt.Println("ldap bind error: ", err)
		return
	}

	m, _ := json.Marshal(sr.Entries[0])
	auth = true
	meta = string(m)
	return
}

func (f *filter) verify(header api.RequestHeaderMap) (bool, string) {
	auth, ok := header.Get("authorization")
	if !ok {
		return false, "no Authorization"
	}
	username, password, ok := parseUsernameAndPassword(auth)
	if !ok {
		return false, "invalid Authorization format"
	}
	fmt.Printf("got username: %v, password: %v\n", username, password)

	ok, meta := authLdap(f.config, username, password)
	fmt.Println("meta: ", meta)
	if !ok {
		return false, "invalid username or password"
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
