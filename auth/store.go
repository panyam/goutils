package auth

// Our store interfaces for where these are all stored

type ChannelStore interface {
	SaveChannel(channel *Channel) *Channel
	EnsureChannel(provider string, loginId string, params map[string]interface{}) (*Channel, bool)
}

type AuthFlowStore interface {
	GetAuthFlowById(authFlowId string) *AuthFlow
	DeleteAuthFlowById(authFlowId string) bool
	/**
	 * Creates a new auth session object to track a login request.
	 */
	SaveAuthFlow(authFlow *AuthFlow) *AuthFlow
}

type IdentityStore interface {
	EnsureIdentity(identityType string, identityKey string, params map[string]interface{}) (*Identity, bool)
}
