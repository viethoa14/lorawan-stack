// Copyright © 2020 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gatewayserver

import (
	"context"

	"go.thethings.network/lorawan-stack/v3/pkg/ttnpb"
	"google.golang.org/grpc"
)

// GatewayConnectionStatsRegistry stores, updates and cleans up gateway connection stats.
type GatewayConnectionStatsRegistry interface {
	// Get returns connection stats for a gateway.
	Get(ctx context.Context, ids ttnpb.GatewayIdentifiers) (*ttnpb.GatewayConnectionStats, error)
	// Set sets or clears the connection stats for a gateway.
	Set(ctx context.Context, ids ttnpb.GatewayIdentifiers, stats *ttnpb.GatewayConnectionStats) error
}

// EntityRegistry abstracts the Identity server gateway functions.
type EntityRegistry interface {
	// AssertGatewayRights checks whether the gateway authentication (provied in the context) contains the required rights.
	AssertGatewayRights(ctx context.Context, ids ttnpb.GatewayIdentifiers, required ...ttnpb.Right) error
	// Get the identifiers of the gateway that has the given EUI registered.
	GetIdentifiersForEUI(ctx context.Context, in *ttnpb.GetGatewayIdentifiersForEUIRequest, opts ...grpc.CallOption) (*ttnpb.GatewayIdentifiers, error)
	// Get the gateway with the given identifiers, selecting the fields specified.
	Get(ctx context.Context, in *ttnpb.GetGatewayRequest, opts ...grpc.CallOption) (*ttnpb.Gateway, error)
	// Update the gateway, changing the fields specified by the field mask to the provided values.
	Update(ctx context.Context, in *ttnpb.UpdateGatewayRequest, opts ...grpc.CallOption) (*ttnpb.Gateway, error)
}
