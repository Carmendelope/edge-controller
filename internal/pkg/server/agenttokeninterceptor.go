package server

import (
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/rs/zerolog/log"
	"strings"
)

type AgentTokenInterceptor struct {
	tokenProvider asset.Provider
}

func NewAgentTokenInterceptor (provider asset.Provider) *AgentTokenInterceptor {
	return &AgentTokenInterceptor{
		tokenProvider: provider,
	}
}

func (at *AgentTokenInterceptor) Connect() derrors.Error {
	return nil
}


// validJoinToken
func (at *AgentTokenInterceptor) validJoinToken(organizationID string, token string)  derrors.Error {

	check, err := at.tokenProvider.CheckJoinToken(token)
	if err != nil {
		return  err
	}
	if check == false {
		return derrors.NewUnauthenticatedError("invalid join token")
	}

	return nil
}

// validAgentToken
func (at *AgentTokenInterceptor) validAgentToken(organizationID string, assetID string, token string) derrors.Error {

	asset, err := at.tokenProvider.GetAssetByToken(token)
	if err != nil {
		return  err
	}
	if asset.AssetId != assetID {
		return derrors.NewUnauthenticatedError("invalid agent token")
	}

	return nil
}

// IsValid First check if the token is valid
// AgentToken: has three fields separated by # (organization_id#asset_id#token)
// JoinToken: has two fields separated by # (organization_id#token)
func (at *AgentTokenInterceptor) IsValid (tokenInfo string) derrors.Error {

	splitToken := strings.Split(tokenInfo, "#")

	if len(splitToken) != 2 && len(splitToken) != 3{
		log.Warn().Str("tokenInfo", tokenInfo).Msg("cannot validate token. Error in token format")
		return derrors.NewUnauthenticatedError("cannot validate token")
	}
	// JoinToken
	if len(splitToken) == 2 {
		at.validJoinToken(splitToken[0], splitToken[1])
	}
	if len(splitToken) == 3 {
		return at.validAgentToken(splitToken[0], splitToken[1], splitToken[2])
	}
	return nil
}