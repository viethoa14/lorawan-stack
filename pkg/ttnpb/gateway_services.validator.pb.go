// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: lorawan-stack/api/gateway_services.proto

package ttnpb // import "go.thethings.network/lorawan-stack/pkg/ttnpb"

import github_com_mwitkow_go_proto_validators "github.com/mwitkow/go-proto-validators"
import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/gogo/protobuf/gogoproto"
import _ "github.com/golang/protobuf/ptypes/empty"
import _ "google.golang.org/genproto/googleapis/api/annotations"
import _ "google.golang.org/genproto/protobuf/field_mask"

import time "time"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

func (this *PullGatewayConfigurationRequest) Validate() error {
	if err := github_com_mwitkow_go_proto_validators.CallValidatorIfExists(&(this.GatewayIdentifiers)); err != nil {
		return github_com_mwitkow_go_proto_validators.FieldError("GatewayIdentifiers", err)
	}
	if err := github_com_mwitkow_go_proto_validators.CallValidatorIfExists(&(this.FieldMask)); err != nil {
		return github_com_mwitkow_go_proto_validators.FieldError("FieldMask", err)
	}
	return nil
}
