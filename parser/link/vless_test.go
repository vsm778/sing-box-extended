package link

import (
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestParseSubscriptionLinkVLESSREALITYMLKEM(t *testing.T) {
	// Synthetic credentials and a documentation address; no network access is needed.
	const baseLink = "vless://00000000-0000-4000-8000-000000000000@192.0.2.1:443?security=reality&fp=chrome&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&sid=00"

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "absent"},
		{name: "true", query: "support-x25519mlkem768=true", want: true},
		{name: "false", query: "support-x25519mlkem768=false"},
		{name: "one", query: "support-x25519mlkem768=1", want: true},
		{name: "zero", query: "support-x25519mlkem768=0"},
		{name: "uppercase", query: "support-x25519mlkem768=TRUE", want: true},
		{name: "underscore_true", query: "support_x25519mlkem768=true", want: true},
		{name: "underscore_false", query: "support_x25519mlkem768=false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := baseLink
			if tt.query != "" {
				link += "&" + tt.query
			}
			outbound, err := ParseSubscriptionLink(link)
			if err != nil {
				t.Fatal(err)
			}
			options, ok := outbound.Options.(*option.VLESSOutboundOptions)
			if !ok {
				t.Fatalf("unexpected outbound options: %T", outbound.Options)
			}
			if options.TLS == nil || options.TLS.Reality == nil {
				t.Fatal("missing REALITY TLS options")
			}
			if got := options.TLS.Reality.SupportX25519MLKEM768; got != tt.want {
				t.Errorf("SupportX25519MLKEM768 = %v, want %v", got, tt.want)
			}
		})
	}
}
