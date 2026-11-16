package link

import (
	"encoding/json"
	"strconv"

	"github.com/sagernet/sing-box/common"
	"github.com/sagernet/sing-box/option"
)

// Xray URI extras use camelCase; native sing-box options use snake_case.
// Native keys win over aliases, and sessionPlacement wins over sessionIDPlacement.
var xhttpExtraAliases = [][2]string{
	{"noGRPCHeader", "no_grpc_header"},
	{"noSSEHeader", "no_sse_header"},
	{"xPaddingBytes", "x_padding_bytes"},
	{"xPaddingObfsMode", "x_padding_obfs_mode"},
	{"xPaddingKey", "x_padding_key"},
	{"xPaddingHeader", "x_padding_header"},
	{"xPaddingPlacement", "x_padding_placement"},
	{"xPaddingMethod", "x_padding_method"},
	{"uplinkHTTPMethod", "uplink_http_method"},
	{"sessionPlacement", "session_placement"},
	{"sessionIDPlacement", "session_placement"},
	{"sessionKey", "session_key"},
	{"sessionIDKey", "session_key"},
	{"seqPlacement", "seq_placement"},
	{"seqKey", "seq_key"},
	{"uplinkDataPlacement", "uplink_data_placement"},
	{"uplinkDataKey", "uplink_data_key"},
	{"uplinkChunkSize", "uplink_chunk_size"},
	{"scMaxEachPostBytes", "sc_max_each_post_bytes"},
	{"scMinPostsIntervalMs", "sc_min_posts_interval_ms"},
	{"scMaxBufferedPosts", "sc_max_buffered_posts"},
	{"scStreamUpServerSecs", "sc_stream_up_server_secs"},
	{"serverMaxHeaderBytes", "server_max_header_bytes"},
	{"sessionIDTable", "session_id_table"},
	{"sessionIDLength", "session_id_length"},
	{"congestionController", "congestion_controller"},
}

var xhttpXmuxAliases = [][2]string{
	{"maxConcurrency", "max_concurrency"},
	{"maxConnections", "max_connections"},
	{"cMaxReuseTimes", "c_max_reuse_times"},
	{"hMaxRequestTimes", "h_max_request_times"},
	{"hMaxReusableSecs", "h_max_reusable_secs"},
	{"hKeepAlivePeriod", "h_keep_alive_period"},
}

func normalizeXHTTPExtraKeys(fields map[string]json.RawMessage, aliases [][2]string) {
	for _, alias := range aliases {
		if value, exists := fields[alias[0]]; exists {
			if _, canonicalExists := fields[alias[1]]; !canonicalExists {
				fields[alias[1]] = value
			}
			delete(fields, alias[0])
		}
	}
}

func normalizeXHTTPExtraIntegers(fields map[string]json.RawMessage, keys ...string) error {
	for _, key := range keys {
		var value string
		if raw, exists := fields[key]; exists && json.Unmarshal(raw, &value) == nil {
			number, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			fields[key] = json.RawMessage(strconv.FormatInt(number, 10))
		}
	}
	return nil
}

func applyVLESSXHTTPExtra(extra string, options *option.V2RayXHTTPOptions) error {
	// The shared decoder accepts raw JSON as well as padded/unpadded base64.
	decoded, err := common.DecodeBase64URLSafe(extra)
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(decoded), &fields); err != nil {
		return err
	}
	normalizeXHTTPExtraKeys(fields, xhttpExtraAliases)
	if err := normalizeXHTTPExtraIntegers(fields, "sc_max_buffered_posts", "server_max_header_bytes", "cwnd"); err != nil {
		return err
	}
	if raw, exists := fields["xmux"]; exists {
		var xmux map[string]json.RawMessage
		if err := json.Unmarshal(raw, &xmux); err != nil {
			return err
		}
		normalizeXHTTPExtraKeys(xmux, xhttpXmuxAliases)
		if err := normalizeXHTTPExtraIntegers(xmux, "h_keep_alive_period"); err != nil {
			return err
		}
		fields["xmux"], err = json.Marshal(xmux)
		if err != nil {
			return err
		}
	}
	data, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	// Use the native field types (including numeric and string ranges), but keep
	// the query's existing host/path/mode when the extra does not supply them.
	// Full transport validation remains with the normal config loader.
	type plainXHTTPOptions option.V2RayXHTTPOptions
	return json.Unmarshal(data, (*plainXHTTPOptions)(options))
}
