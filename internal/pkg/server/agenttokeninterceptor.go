package server

import (
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/provider/asset"
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
func (at *AgentTokenInterceptor) validJoinToken(token string)  derrors.Error {

	check, err := at.tokenProvider.CheckJoinToken(token)
	if err != nil {
		return  err
	}
	if check == false {
		return derrors.NewUnauthenticatedError("invalid token")
	}

	return nil
}

// validAgentToken checks if the token
func (at *AgentTokenInterceptor) validAgentToken(token string) derrors.Error {

	_, err := at.tokenProvider.GetAssetByToken(token)
	if err != nil {
		return  err
	}

	return nil
}

// IsValid First check if the token is valid ( First check if the token is a valid agent token, if not check if it is a valid join token)
func (at *AgentTokenInterceptor) IsValid (tokenInfo string) derrors.Error {

	// check if is a valid agent token
	err := at.validAgentToken(tokenInfo)
	// if not check if it is a valid join token
	if err != nil {
		return at.validJoinToken(tokenInfo)

	}
	return nil
}