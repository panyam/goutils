package auth

// Our store interfaces for where these are all stored
type EnsuredItem[T any] struct {
	Value      T
	NewCreated bool
}

type ChannelStore interface {
	SaveChannel(channel *Channel) chan *Channel
	EnsureChannel(provider string, loginId string, params map[string]interface{}) chan EnsuredItem[*Channel]
}

type AuthFlowStore interface {
	GetAuthFlowById(authFlowId string) chan *AuthFlow
	DeleteAuthFlowById(authFlowId string) chan bool
	/**
	 * Creates a new auth session object to track a login request.
	 */
	SaveAuthFlow(authFlow *AuthFlow) chan *AuthFlow
}

type IdentityStore interface {
	EnsureIdentity(identityType string, identityKey string, params map[string]interface{}) chan EnsuredItem[*Identity]
}
