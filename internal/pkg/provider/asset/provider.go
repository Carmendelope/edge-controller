/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package asset

import (
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"time"
)

// TTL for agent join tokens.
const AgentJoinTokenTTL = time.Hour

type Provider interface {

	// AddECOpResponse stores a response for an operation executed by the edge controller.
	AddECOpResponse(op entities.EdgeControllerOpResponse) derrors.Error
	// GetPendingECOpResponses retrieves the list of pending operation responses
	GetPendingECOpResponses(removeEntries bool)([]entities.EdgeControllerOpResponse, derrors.Error)

	// AddPendingOperation stores a pending operation for an agent.
	AddPendingOperation(op entities.AgentOpRequest) derrors.Error
	// GetPendingOperations retrieves the list of pending operations for a given asset. The removeEntries
	// flags determines if the elements are removed before returning the list.
	GetPendingOperations(assetID string, removeEntries bool) ([]entities.AgentOpRequest, derrors.Error)

	// AddPendingOperationResult stores a pending operation for an agent.
	AddOpResponse(op entities.AgentOpResponse) derrors.Error
	// GetPendingOpResponses retrieves the list of pending operation responses
	GetPendingOpResponses(removeEntries bool)([]entities.AgentOpResponse, derrors.Error)

	// AddAgentStart stores a pending message with the agent start information.
	AddAgentStart(op entities.AgentStartInfo) derrors.Error
	// GetPendingAgetStart retrieves the list of Agent start operations that need to be send
	GetPendingAgentStart(removeEntries bool) ([]entities.AgentStartInfo, derrors.Error)

	// AddManagedAsset adds a new asset to the list of assets that are managed by this EIC and can send data to it.
	AddManagedAsset(asset entities.AgentJoinInfo) derrors.Error
	// RemoveManagedAsset removes an asset from the list.
	RemoveManagedAsset(assetID string) derrors.Error
	// GetAssetByToken checks if there is an asset with a given token.
	GetAssetByToken(token string) (*entities.AgentJoinInfo, derrors.Error)
	// AddJoinToken adds a new join token for agents
	AddJoinToken(joinToken string) (*entities.JoinToken, derrors.Error)
	// CheckJoinToken checks if a join token is valid
	CheckJoinToken(joinToken string) (bool, derrors.Error)
	// Clear all elements
	Clear() derrors.Error
}