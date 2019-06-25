package eic

import (
	"github.com/nalej/edge-controller/internal/pkg/server/agent"
	grpc_inventory_manager_go "github.com/nalej/grpc-inventory-manager-go"
	"github.com/rs/zerolog/log"
)

type AgentInstaller struct{
	notifier *agent.Notifier
}

func NewAgentInstaller(notifier *agent.Notifier) * AgentInstaller{
	return &AgentInstaller{notifier}
}

func (ai * AgentInstaller) InstallAgent(operationID string, request * grpc_inventory_manager_go.InstallAgentRequest){
	log.Debug().Interface("request", request).Msg("triggering agent install")

}

func (ai * AgentInstaller) copyBinaryToAsset(operationID string, request * grpc_inventory_manager_go.InstallAgentRequest) {

}