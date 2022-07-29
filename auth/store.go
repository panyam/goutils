package auth

import (
	"github.com/panyam/goutils/utils"
)

// Our store interfaces for where these are all stored

type ChannelStore interface {
	SaveChannel(channel *Channel) error
	EnsureChannel(provider string, loginId string, params utils.StringMap) (*Channel, bool)
}

type AuthFlowStore interface {
	GetAuthFlowById(authFlowId string) (*AuthFlow, error)
	DeleteAuthFlowById(authFlowId string) bool
	/**
	 * Creates a new auth session object to track a login request.
	 */
	SaveAuthFlow(authFlow *AuthFlow) *AuthFlow
}

type IdentityStore interface {
	EnsureIdentity(identityType string, identityKey string, params utils.StringMap) (*Identity, bool)
}
