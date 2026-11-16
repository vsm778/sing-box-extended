package link

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func parseTestVLESSXHTTP(t *testing.T, extra string) *option.V2RayTransportOptions {
	t.Helper()
	// Documentation-only endpoint and synthetic credentials. No network access.
	query := url.Values{
		"security": {"tls"}, "sni": {"example.com"}, "fp": {"chrome"},
		"type": {"xhttp"}, "host": {"example.com"}, "path": {"/test/"},
		"mode": {"packet-up"}, "x_padding_bytes": {"100-200"},
	}
	if extra != "" {
		query.Set("extra", extra)
	}
	outbound, err := ParseSubscriptionLink("vless://00000000-0000-4000-8000-000000000000@192.0.2.1:443?" + query.Encode())
	if err != nil {
		t.Fatal(err)
	}
	options, ok := outbound.Options.(*option.VLESSOutboundOptions)
	if !ok || options.Transport == nil || options.Transport.Type != "xhttp" {
		t.Fatalf("missing VLESS XHTTP transport: %T", outbound.Options)
	}
	if options.TLS == nil || !options.TLS.Enabled || options.TLS.ServerName != "example.com" {
		t.Fatal("TLS parameters were not preserved")
	}
	return options.Transport
}

func TestParseSubscriptionLinkVLESSXHTTPExtra(t *testing.T) {
	const extra = `{"mode":"packet-up","uplinkHTTPMethod":"GET","uplinkDataPlacement":"header","uplinkDataKey":"X-Data","sessionIDPlacement":"path","seqPlacement":"path","xPaddingBytes":"100-200","xPaddingObfsMode":true,"xPaddingKey":"padding","xPaddingHeader":"X-Pad","xPaddingPlacement":"header","xPaddingMethod":"tokenish","noGRPCHeader":true,"noSSEHeader":true,"scMaxEachPostBytes":4096,"scMinPostsIntervalMs":10,"scStreamUpServerSecs":"20-30","xmux":{"maxConnections":2,"cMaxReuseTimes":"3-4","hMaxRequestTimes":100,"hMaxReusableSecs":"60-120","hKeepAlivePeriod":15}}`
	encodings := []struct {
		name string
		data string
	}{
		{"json", extra},
		{"base64", base64.StdEncoding.EncodeToString([]byte(extra))},
		{"base64_raw", base64.RawStdEncoding.EncodeToString([]byte(extra))},
		{"base64_url", base64.URLEncoding.EncodeToString([]byte(extra))},
		{"base64_url_raw", base64.RawURLEncoding.EncodeToString([]byte(extra))},
	}
	for _, encoding := range encodings {
		t.Run(encoding.name, func(t *testing.T) {
			transport := parseTestVLESSXHTTP(t, encoding.data)
			x := transport.XHTTPOptions
			if x.Host != "example.com" || x.Path != "/test/" || x.Mode != "packet-up" {
				t.Errorf("query options lost: host=%q path=%q mode=%q", x.Host, x.Path, x.Mode)
			}
			if x.UplinkHTTPMethod != "GET" || x.UplinkDataPlacement != "header" || x.UplinkDataKey != "X-Data" {
				t.Errorf("GET/header uplink lost: method=%q placement=%q key=%q", x.UplinkHTTPMethod, x.UplinkDataPlacement, x.UplinkDataKey)
			}
			if x.SessionPlacement != "path" || x.SeqPlacement != "path" {
				t.Errorf("session/sequence placement lost: %q/%q", x.SessionPlacement, x.SeqPlacement)
			}
			if x.XPaddingBytes.From != 100 || x.XPaddingBytes.To != 200 || !x.XPaddingObfsMode || x.XPaddingKey != "padding" || x.XPaddingHeader != "X-Pad" || x.XPaddingPlacement != "header" || x.XPaddingMethod != "tokenish" {
				t.Error("padding options lost")
			}
			if !x.NoGRPCHeader || !x.NoSSEHeader || x.ScMaxEachPostBytes == nil || x.ScMaxEachPostBytes.From != 4096 || x.ScMaxEachPostBytes.To != 4096 || x.ScMinPostsIntervalMs == nil || x.ScMinPostsIntervalMs.From != 10 || x.ScStreamUpServerSecs == nil || x.ScStreamUpServerSecs.To != 30 {
				t.Error("boolean or numeric/range options lost")
			}
			if x.Xmux == nil || x.Xmux.MaxConnections.From != 2 || x.Xmux.CMaxReuseTimes.To != 4 || x.Xmux.HMaxRequestTimes.From != 100 || x.Xmux.HMaxReusableSecs.To != 120 || x.Xmux.HKeepAlivePeriod != 15 {
				t.Error("XMUX options lost")
			}
			// Check that the imported options survive the same JSON validation as a config.
			data, err := json.Marshal(transport)
			if err != nil {
				t.Fatal(err)
			}
			var checked option.V2RayTransportOptions
			if err := json.Unmarshal(data, &checked); err != nil {
				t.Fatalf("imported transport is not a valid config: %v", err)
			}
			if checked.XHTTPOptions.UplinkHTTPMethod != "GET" || checked.XHTTPOptions.UplinkDataPlacement != "header" {
				t.Error("GET/header options changed during JSON round trip")
			}
		})
	}
}

func TestParseSubscriptionLinkVLESSXHTTPSnakeCase(t *testing.T) {
	x := parseTestVLESSXHTTP(t, `{"uplink_http_method":"GET","uplink_data_placement":"header","uplink_data_key":"X-Test","session_placement":"header","session_key":"X-Session","seq_placement":"query","seq_key":"seq","sc_max_each_post_bytes":"2048-4096","x_padding_bytes":128,"xmux":{"max_connections":"2-3","h_keep_alive_period":5},"unknownFutureOption":true}`).XHTTPOptions
	if x.UplinkHTTPMethod != "GET" || x.UplinkDataPlacement != "header" || x.UplinkDataKey != "X-Test" || x.SessionPlacement != "header" || x.SessionKey != "X-Session" || x.SeqPlacement != "query" || x.SeqKey != "seq" {
		t.Error("native snake_case options lost")
	}
	if x.ScMaxEachPostBytes == nil || x.ScMaxEachPostBytes.To != 4096 || x.XPaddingBytes.From != 128 || x.Xmux == nil || x.Xmux.MaxConnections.To != 3 || x.Xmux.HKeepAlivePeriod != 5 {
		t.Error("native range/XMUX options lost")
	}
}

func TestParseSubscriptionLinkVLESSXHTTPAliasPriority(t *testing.T) {
	for i := 0; i < 30; i++ {
		x := parseTestVLESSXHTTP(t, `{"sessionIDPlacement":"query","sessionPlacement":"header","session_placement":"path","uplinkHTTPMethod":"POST","uplink_http_method":"GET"}`).XHTTPOptions
		if x.SessionPlacement != "path" || x.UplinkHTTPMethod != "GET" {
			t.Fatal("canonical snake_case option must win over aliases")
		}
		x = parseTestVLESSXHTTP(t, `{"sessionIDPlacement":"query","sessionPlacement":"header","sessionKey":"X-Session"}`).XHTTPOptions
		if x.SessionPlacement != "header" || x.SessionKey != "X-Session" {
			t.Fatal("sessionPlacement must win over the legacy sessionIDPlacement alias")
		}
	}
}

func TestParseSubscriptionLinkVLESSXHTTPQueryPadding(t *testing.T) {
	x := parseTestVLESSXHTTP(t, "").XHTTPOptions
	if x.XPaddingBytes.From != 100 || x.XPaddingBytes.To != 200 {
		t.Fatal("query x_padding_bytes lost without extra")
	}
}

func TestParseSubscriptionLinkVLESSXHTTPLegacyStrings(t *testing.T) {
	x := parseTestVLESSXHTTP(t, `{"noGRPCHeader":true,"xPaddingBytes":"100-200","scMaxEachPostBytes":"4096","scMinPostsIntervalMs":"10-20","xmux":{"maxConnections":"2-3","hMaxRequestTimes":"100-200","hKeepAlivePeriod":"15"},"headers":{"X-Custom-Header":"test"}}`).XHTTPOptions
	if !x.NoGRPCHeader || x.ScMaxEachPostBytes == nil || x.ScMaxEachPostBytes.From != 4096 || x.ScMinPostsIntervalMs == nil || x.ScMinPostsIntervalMs.To != 20 || x.Xmux == nil || x.Xmux.MaxConnections.To != 3 || x.Xmux.HMaxRequestTimes.To != 200 || x.Xmux.HKeepAlivePeriod != 15 || x.Headers["X-Custom-Header"] != "test" {
		t.Fatal("legacy strings or HTTP header keys changed during extra import")
	}
}

func TestParseSubscriptionLinkVLESSXHTTPInvalidExtra(t *testing.T) {
	for _, extra := range []string{`{`, `[]`, `{"uplinkHTTPMethod":42}`, `{"scMaxEachPostBytes":"invalid"}`, `{"xmux":{"hKeepAlivePeriod":"invalid"}}`} {
		t.Run(extra, func(t *testing.T) {
			query := url.Values{"type": {"xhttp"}, "extra": {extra}}
			_, err := ParseSubscriptionLink("vless://00000000-0000-4000-8000-000000000000@192.0.2.1:443?" + query.Encode())
			if err == nil {
				t.Fatal("malformed extra must be rejected instead of silently losing transport options")
			}
		})
	}
}
