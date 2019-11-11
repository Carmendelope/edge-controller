/*
 * Copyright 2019 Nalej
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agent

import (
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/utils"
	"github.com/nalej/grpc-common-go"
	"github.com/nalej/grpc-edge-controller-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc/peer"
)

// Handler structure for the cluster requests.
type Handler struct {
	Manager Manager
}

// NewHandler creates a new Handler with a linked manager.
func NewHandler(manager Manager) *Handler{
	return &Handler{manager}
}

func (h *Handler) AgentJoin(ctx context.Context, request *grpc_edge_controller_go.AgentJoinRequest) (*grpc_inventory_manager_go.AgentJoinResponse, error) {
	err := entities.ValidJoinRequest(request)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	response, err := h.Manager.AgentJoin(request)
	if err != nil{
		log.Warn().Str("trace", err.DebugReport()).Msg("agent join failed")
		return nil, conversions.ToGRPCError(err)
	}
	log.Debug().Str("assetID", response.AssetId).Msg("agent joined successfully")
	return response, nil
}

func (h *Handler) AgentStart(ctx context.Context, info *grpc_inventory_manager_go.AgentStartInfo) (*grpc_common_go.Success, error) {
	err := entities.ValidAgentStartInfo(info)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	err = h.Manager.AgentStart(info)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	return &grpc_common_go.Success{}, nil
}

func (h *Handler) AgentCheck(ctx context.Context, request *grpc_edge_controller_go.AgentCheckRequest) (*grpc_edge_controller_go.CheckResult, error) {

	err := entities.ValidAgentCheckRequest(request)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}

	// get agent IP
	ip := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		ip = utils.RemovePort(peer.Addr.String())
	}else{
		log.Warn().Str("assetID", request.AssetId).Msg("error getting agent IP")
	}

	result, err := h.Manager.AgentCheck(request, ip)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	return result, nil
}

func (h *Handler) CallbackAgentOperation(ctx context.Context, response *grpc_inventory_manager_go.AgentOpResponse) (*grpc_common_go.Success, error) {
	log.Info().Interface("response", response).Msg("Callback agent operation")
	err := entities.ValidAgentOpResponse(response)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	err = h.Manager.CallbackAgentOperation(response)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	return &grpc_common_go.Success{}, nil
}

