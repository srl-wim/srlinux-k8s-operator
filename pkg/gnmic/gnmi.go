package gnmiclient

import (
	"context"
	"fmt"

	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/metadata"
)

// Set sends a gnmi.SetRequest to the target *t and returns a gnmi.SetResponse and an error
func (g *GnmiClient) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	nctx, cancel := context.WithTimeout(ctx, g.Timeout)
	defer cancel()
	nctx = metadata.AppendToOutgoingContext(nctx, "username", g.Username, "password", g.Password)
	response, err := g.Client.Set(nctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed sending SetRequest to '%s': %v", g.Target, err)
	}
	return response, nil
}

// Get sends a gnmi.GetRequest to the target *t and returns a gnmi.GetResponse and an error
func (g *GnmiClient) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	nctx, cancel := context.WithTimeout(ctx, g.Timeout)
	defer cancel()
	nctx = metadata.AppendToOutgoingContext(nctx, "username", g.Username, "password", g.Password)
	response, err := g.Client.Get(nctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed sending GetRequest to '%s': %v", g.Target, err)
	}
	return response, nil
}
