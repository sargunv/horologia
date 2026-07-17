package api

import (
	"context"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
)

// ServerInfoGet reports the compatibility contract required by native clients.
func (h *Handler) ServerInfoGet(context.Context) (*apigen.ServerInfoResponse, error) {
	return &apigen.ServerInfoResponse{
		ApiVersion: apigen.ServerInfoResponseApiVersion1,
		Capabilities: []apigen.ServerCapability{
			apigen.ServerCapabilityOAuth21Pkce,
			apigen.ServerCapabilityWidgetSnapshotsV1,
		},
	}, nil
}
