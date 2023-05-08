package envoy_ldap_go

import (
	"encoding/base64"
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

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
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

// newLdapClient creates a new ldap client.
func newLdapClient(config *config) (*ldap.Conn, error) {
	client, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", config.host, config.port))
	if err != nil {
		return nil, err
	}
	err = client.Bind(config.username, config.password)
	// First bind with a read only user
	if err != nil {
		return nil, err
	}
	return client, err
}

// authLdap authenticates the user against the ldap server.
func authLdap(config *config, username, password string) (auth bool, meta string) {
	client, err := newLdapClient(config)
	if err != nil {
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
		fmt.Sprintf("(sAMAccountName=%s)", username),
		[]string{"dn", "cn"}, nil)
	sr, err := client.Search(req)
	if err != nil {
		return
	}
	if len(sr.Entries) != 1 {
		return
	}
	userdn := sr.Entries[0].DN
	err = client.Bind(userdn, password)
	if err != nil {
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
	username, password, ok := parseBasicAuth(auth)
	if !ok {
		return false, "invalid Authorization format"
	}
	fmt.Printf("got username: %v, password: %v\n", username, password)

	if ok, _ := authLdap(f.config, username, password); !ok {
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
