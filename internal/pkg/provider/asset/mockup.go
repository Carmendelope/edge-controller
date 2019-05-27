/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package asset

import (
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"sync"
	"time"
)

type MockupAssetProvider struct {
	// Mutex for managing mockup access.
	sync.Mutex
	// AssetsByAssetID with a map of assets indexed by assetID.
	assetsByAssetID map[string]entities.AgentJoinInfo
	// AssetsByToken with a map of assets indexed by token.
	assetsByToken map[string]entities.AgentJoinInfo
	// pendingOps with a map of operations pending per asset identifier.
	pendingOps map[string][]entities.AgentOpRequest
	// pendingResult with a map of pending responses per operation identifier.
	pendingResult map[string]entities.AgentOpResponse
	// joinToken map with the join tokens and their expiration date.
	joinToken map[string]int64
	// agentStart map with the start info by asset identifier.
	agentStart map[string]entities.AgentStartInfo
}

func NewMockupAssetProvider() * MockupAssetProvider{
	return &MockupAssetProvider{
		assetsByAssetID: make(map[string]entities.AgentJoinInfo, 0),
		assetsByToken: make(map[string]entities.AgentJoinInfo, 0),
		pendingOps: make(map[string][]entities.AgentOpRequest, 0),
		pendingResult: make(map[string]entities.AgentOpResponse, 0),
		joinToken: make(map[string]int64, 0),
		agentStart: make(map[string]entities.AgentStartInfo, 0),
	}
}

func (m*MockupAssetProvider) unsafeExistAsset(assetID string) bool{
	_, exists := m.assetsByAssetID[assetID]
	return exists
}

func (m*MockupAssetProvider) unsafeExistJoinToken(joinToken string) bool{
	_, exists := m.joinToken[joinToken]
	return exists
}

func (m*MockupAssetProvider) unsafeExistAgentStart(assetID string) bool{
	_, exists := m.agentStart[assetID]
	return exists
}

func (m *MockupAssetProvider) AddPendingOperation(op entities.AgentOpRequest) derrors.Error {
	m.Lock()
	defer m.Unlock()
	if !m.unsafeExistAsset(op.AssetId){
		return derrors.NewFailedPreconditionError("asset is not managed by this EIC").WithParams(op.AssetId)
	}
	opList, _ := m.pendingOps[op.AssetId]
	m.pendingOps[op.AssetId] = append(opList, op)
	return nil
}

func (m *MockupAssetProvider) GetPendingOperations(assetID string, removeEntries bool) ([]entities.AgentOpRequest, derrors.Error) {
	m.Lock()
	defer m.Unlock()
	if !m.unsafeExistAsset(assetID){
		return nil, derrors.NewFailedPreconditionError("asset is not managed by this EIC").WithParams(assetID)
	}
	opList, exist := m.pendingOps[assetID]
	if !exist{
		return make([]entities.AgentOpRequest, 0), nil
	}
	if removeEntries{
		delete(m.pendingOps, assetID)
	}
	return opList, nil
}

// AddPendingOperationResult stores a pending operation for an agent.
func (m *MockupAssetProvider) AddOpResponse(op entities.AgentOpResponse) derrors.Error{
	m.Lock()
	defer m.Unlock()
	_, exists := m.pendingResult[op.OperationId]
	if exists{
		return derrors.NewAlreadyExistsError("operation result already registered").WithParams(op.OperationId)
	}
	m.pendingResult[op.OperationId] = op
	return nil
}

// GetPendingOpResponses retrieves the list of pending operation responses
func (m *MockupAssetProvider) GetPendingOpResponses(removeEntries bool)([]entities.AgentOpResponse, derrors.Error){
	m.Lock()
	defer m.Unlock()
	result := make([]entities.AgentOpResponse, 0, len(m.agentStart))
	for _, v := range m.pendingResult{
		result = append(result, v)
	}
	if removeEntries{
		m.pendingResult = make(map[string]entities.AgentOpResponse, 0)
	}
	return result, nil
}

// AddAgentStart stores a pending message with the agent start information.
func (m *MockupAssetProvider) AddAgentStart(op entities.AgentStartInfo) derrors.Error{
	m.Lock()
	defer m.Unlock()
	m.agentStart[op.AssetId] = op
	return nil
}

// GetPendingAgetStart retrieves the list of Agent start operations that need to be send
func (m *MockupAssetProvider) GetPendingAgentStart(removeEntries bool) ([]entities.AgentStartInfo, derrors.Error){
	m.Lock()
	defer m.Unlock()
	result := make([]entities.AgentStartInfo, 0, len(m.agentStart))
	for _, v := range m.agentStart{
		result = append(result, v)
	}
	if removeEntries{
		m.agentStart = make(map[string]entities.AgentStartInfo, 0)
	}
	return result, nil
}


func (m *MockupAssetProvider) AddManagedAsset(asset entities.AgentJoinInfo) derrors.Error {
	m.Lock()
	defer m.Unlock()
	if m.unsafeExistAsset(asset.AssetId){
		return derrors.NewAlreadyExistsError("asset already managed by this EIC").WithParams(asset.AssetId)
	}
	m.assetsByToken[asset.Token] = asset
	m.assetsByAssetID[asset.AssetId] = asset
	return nil
}

func (m *MockupAssetProvider) RemoveManagedAsset(assetID string) derrors.Error {
	m.Lock()
	defer m.Unlock()
	if !m.unsafeExistAsset(assetID){
		return derrors.NewFailedPreconditionError("asset is not managed by this EIC").WithParams(assetID)
	}
	associatedToken := ""
	for token, asset := range m.assetsByToken{
		if asset.AssetId == assetID {
			associatedToken = token
			break
		}
	}
	delete(m.assetsByToken, associatedToken)
	delete(m.assetsByAssetID, assetID)
	return nil
}

func (m *MockupAssetProvider) GetAssetByToken(token string) (*entities.AgentJoinInfo, derrors.Error) {
	m.Lock()
	defer m.Unlock()
	asset, exists := m.assetsByToken[token]
	if !exists{
		return nil, derrors.NewFailedPreconditionError("asset is not managed by this EIC")
	}
	return &asset, nil
}

// AddJoinToken adds a new join token for agents
func (m *MockupAssetProvider) AddJoinToken(joinToken string) (*entities.JoinToken, derrors.Error){
	m.Lock()
	defer m.Unlock()
	if m.unsafeExistJoinToken(joinToken){
		return nil, derrors.NewAlreadyExistsError("agent join token already exists")
	}

	expired := time.Now().Add(AgentJoinTokenTTL).Unix()
	m.joinToken[joinToken] = expired

	return &entities.JoinToken{Token:joinToken, ExpiredOn:expired }, nil
}

// CheckJoinToken checks if a join token is valid
func (m *MockupAssetProvider) CheckJoinToken(joinToken string) (bool, derrors.Error){
	m.Lock()
	defer m.Unlock()
	expire, exists := m.joinToken[joinToken]
	if exists{
		if expire >= time.Now().Unix(){
			return true, nil
		}else{
			// Expire the token
			delete(m.joinToken, joinToken)
		}
	}
	return false, nil
}

// Clear all elements
func (m *MockupAssetProvider) Clear() derrors.Error{
	m.Lock()
	m.assetsByAssetID = make(map[string]entities.AgentJoinInfo, 0)
	m.assetsByToken = make(map[string]entities.AgentJoinInfo, 0)
	m.pendingOps = make(map[string][]entities.AgentOpRequest, 0)
	m.pendingResult = make(map[string]entities.AgentOpResponse, 0)
	m.joinToken = make(map[string]int64, 0)
	m.agentStart = make(map[string]entities.AgentStartInfo, 0)
	m.Unlock()
	return nil
}
