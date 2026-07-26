package webserver

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/voidluo/trojan-go/internal/database"
)

// ctxWithDomain builds a gin context carrying the given X-Node-Domain header.
func ctxWithDomain(header string) *gin.Context {
	return ctxWithIdentity(header, "")
}

func ctxWithIdentity(domain, location string) *gin.Context {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	if domain != "" {
		c.Request.Header.Set("X-Node-Domain", domain)
	}
	if location != "" {
		encoded := base64.RawURLEncoding.EncodeToString([]byte(location))
		c.Request.Header.Set("X-Node-Location-B64", encoded)
	}
	return c
}

func TestReportedNodeDomainAccepted(t *testing.T) {
	cases := map[string]string{
		"xjp.liteops.top":   "xjp.liteops.top",
		"XJP.LiteOps.Top":   "xjp.liteops.top", // normalized to lower case
		"  jp.liteops.top ": "jp.liteops.top",  // surrounding space trimmed
		"a-b.example.com":   "a-b.example.com",
	}
	for header, want := range cases {
		if got := reportedNodeDomain(ctxWithDomain(header)); got != want {
			t.Errorf("reportedNodeDomain(%q) = %q, want %q", header, got, want)
		}
	}
}

// TestReportedNodeDomainRejected guards the values that must never reach
// nodes.address / nodes.sni, because both are published verbatim into
// subscriptions as the Clash `server` and the TLS SNI.
func TestReportedNodeDomainRejected(t *testing.T) {
	cases := []string{
		"",                         // absent header
		"43.160.204.91",            // bare IPv4 — the bug this fix addresses
		"::1",                      // bare IPv6
		"localhost",                // single label, not a certificate domain
		"https://jp.example.com",   // scheme
		"jp.example.com:443",       // port
		"jp.example.com/path",      // path
		"jp.example.com\nInjected", // newline injection
		"jp example.com",           // whitespace
		".example.com",             // leading dot
		"example.com.",             // trailing dot
		"-example.com",             // leading hyphen
		"example.com-",             // trailing hyphen
		"a..example.com",           // empty label
		"jp.exa\u00e4mple.com",     // non-ASCII
	}
	for _, header := range cases {
		if got := reportedNodeDomain(ctxWithDomain(header)); got != "" {
			t.Errorf("reportedNodeDomain(%q) = %q, want empty", header, got)
		}
	}
}

func TestReportedNodeLocationBase64UTF8(t *testing.T) {
	for _, want := range []string{"新加坡", "日本", "美国-洛杉矶"} {
		if got := reportedNodeLocation(ctxWithIdentity("", want)); got != want {
			t.Errorf("reportedNodeLocation = %q, want %q", got, want)
		}
	}

	c := ctxWithDomain("")
	c.Request.Header.Set("X-Node-Location-B64", "%%%not-base64%%")
	if got := reportedNodeLocation(c); got != "" {
		t.Errorf("invalid base64 location = %q, want empty", got)
	}
}

func TestApplyReportedNodeDomainHealsIPValues(t *testing.T) {
	// Pre-fix auto-registration stored the source IP in all three fields.
	node := &database.Node{
		Name:    "43.160.204.91",
		Address: "43.160.204.91",
		SNI:     "43.160.204.91",
	}
	if !applyReportedNodeDomain(node, ctxWithIdentity("xjp.liteops.top", "新加坡")) {
		t.Fatal("expected applyReportedNodeDomain to report a change")
	}
	if node.Address != "xjp.liteops.top" {
		t.Errorf("Address = %q, want xjp.liteops.top", node.Address)
	}
	if node.SNI != "xjp.liteops.top" {
		t.Errorf("SNI = %q, want xjp.liteops.top", node.SNI)
	}
	if node.Name != "新加坡" {
		t.Errorf("Name = %q, want 新加坡", node.Name)
	}
}

func TestApplyReportedNodeDomainFillsEmptyValues(t *testing.T) {
	node := &database.Node{}
	if !applyReportedNodeDomain(node, ctxWithDomain("xjp.liteops.top")) {
		t.Fatal("expected a change for empty node fields")
	}
	if node.Address != "xjp.liteops.top" || node.SNI != "xjp.liteops.top" {
		t.Errorf("empty fields not filled: address=%q sni=%q", node.Address, node.SNI)
	}
}

// TestApplyReportedNodeDomainPreservesOperatorValues is the important guard:
// an operator may deliberately point a node at a specific hostname (relay
// entry/exit setups rely on address != sni), so a domain-shaped value must
// never be overwritten by the worker's self-report.
func TestApplyReportedNodeDomainPreservesOperatorValues(t *testing.T) {
	node := &database.Node{
		Name:    "新加坡(日本转)",
		Address: "jp.liteops.top",  // relay entry
		SNI:     "xjp.liteops.top", // relay exit
	}
	if applyReportedNodeDomain(node, ctxWithDomain("xjp.liteops.top")) {
		t.Error("operator-configured values must not be modified")
	}
	if node.Address != "jp.liteops.top" {
		t.Errorf("Address changed to %q, want jp.liteops.top", node.Address)
	}
	if node.Name != "新加坡(日本转)" {
		t.Errorf("Name changed to %q", node.Name)
	}
}

func TestApplyReportedNodeDomainNoopWithoutHeader(t *testing.T) {
	node := &database.Node{Name: "1.2.3.4", Address: "1.2.3.4", SNI: "1.2.3.4"}
	if applyReportedNodeDomain(node, ctxWithDomain("")) {
		t.Error("missing header must not change the node")
	}
	if node.Address != "1.2.3.4" {
		t.Errorf("Address = %q, want unchanged", node.Address)
	}
}
