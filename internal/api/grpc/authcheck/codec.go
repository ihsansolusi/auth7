package authcheck

import (
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/encoding"
)

// init registers the JSON codec so this gRPC server can decode/encode the
// "json" content-subtype used by lib7-service-go/auth7grpc clients. This mirrors
// that client's codec exactly so auth7 serves the same wire contract WITHOUT
// protobuf code generation.
func init() {
	encoding.RegisterCodec(jsonCodec{})
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("authcheck: marshal: %w", err)
	}
	return data, nil
}

func (jsonCodec) Unmarshal(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("authcheck: unmarshal: %w", err)
	}
	return nil
}

func (jsonCodec) Name() string { return "json" }

var _ encoding.Codec = jsonCodec{}
