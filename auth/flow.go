package auth

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
)

type IdentityInterface struct {
	ChannelId    string
	IdentityType string
	IdentityKey  string
}

type RequestHandler = func(ctx *gin.Context)
type AuthFlowCallback = func(authFlow *AuthFlow, ctx *gin.Context)
type AuthFlowCredentialsReceived = func(ctx *gin.Context)
type IdentityFromProfileFunc = func(profile map[string]interface{}) IdentityInterface
type AuthFlowContinueFunc = func(authFlow *AuthFlow, ctx *gin.Context)
type AuthFlowIdentityEnsured = func(channel *Channel, identity *Identity)

type EnsureLoginConfig struct {
	GetRedirURL   func(ctx *gin.Context) string
	UserParamName string
}

func DefaultEnsureLoginConfig() *EnsureLoginConfig {
	return &EnsureLoginConfig{
		GetRedirURL:   func(ctx *gin.Context) string { return "/auth/login" },
		UserParamName: "loggedInUser",
	}
}

func encodeURIComponent(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	return r
}

/**
 * Redirects users to login screen of they are not logged in
 * @param req Request object
 * @param res Response object
 * @param next next function
 */
func (auth *Authenticator) EnsureLogin(config *EnsureLoginConfig, wrapped RequestHandler) RequestHandler {
	if config != nil {
		config = DefaultEnsureLoginConfig()
	}
	return func(ctx *gin.Context) {
		session := sessions.Default(ctx)
		userParam := session.Get(config.UserParamName)
		if userParam == "" || userParam == nil {
			// Redirect to a login if user not logged in
			// `/${config.redirectURLPrefix || "auth"}/login?callbackURL=${encodeURIComponent(req.originalUrl)}`;
			redirUrl := config.GetRedirURL(ctx)
			originalUrl := ctx.Request.URL.Path
			encodedUrl := encodeURIComponent(originalUrl)
			fullRedirUrl := fmt.Sprintf("%s?callback?callbackURL=%s", redirUrl, encodedUrl)
			ctx.Redirect(http.StatusFound, fullRedirUrl)
		} else {
			wrapped(ctx)
		}
	}
}

type Authenticator struct {
	Provider      string
	ChannelStore  ChannelStore
	IdentityStore IdentityStore
	AuthFlowStore AuthFlowStore

	/**
	 * Step 1. This kicks off the actual auth where credentials can be
	 * extracted from the user.
	 */
	OnAuthFlowStarted AuthFlowCallback

	/**
	 * Step 2.  Once the auth flow is started (eg via a redirect to the provider),
	 * the provider can either redirect to the callback URL (or the user can submit
	 * credentials resulting in a post to this handler).  This gives us a chance to
	 * verify the credentials (and the caller/provider) and redirect success or
	 * failure as necessary.
	 */
	OnAuthFlowCredentialsReceived AuthFlowCredentialsReceived

	/**
	 * Step 3. If credential verification was a success then the provider returns
	 * valid tokens and profile information for us to use/extract.  This method
	 * allows us to extract the channel and identity information from this payload.
	 * This channel ID could be just the user's identity ID too (email, phone etc).
	 */
	IdentityFromProfile IdentityFromProfileFunc

	/**
	 * Step 4. In the previous step the channel and identity information that is
	 * extracted can be processed here by creating new User objects or associating
	 * with existing User objects as is fit within the logic of the application.
	 */
	OnIdentityEnsured AuthFlowIdentityEnsured

	/**
	 * Step 5. Finally after all auth is complete and successful, the continueAuthFlow
	 * callback allows the user to resume the original purpose of the auth flow.
	 * If this method is not provided then a simple redirect to "/" is performed.
	 */
	ContinueAuthFlow AuthFlowContinueFunc
}

/**
 * Step 1 - Called to initiate the auth flow.
 */
func (auth *Authenticator) StartAuthFlow(ctx *gin.Context) {
	q := ctx.Request.URL.Query()
	authFlowId := q["authFlow"][0]
	callbackURL := q["callbackURL"][0]
	authFlow := auth.EnsureAuthFlow(authFlowId, callbackURL)
	if authFlow == nil {
		// Invalid request as auth session is invalid or
		// does not match provider
		ctx.JSON(http.StatusForbidden, gin.H{"error": "NotAllowed", "message": ""})
	} else {
		// Save the auth flow so we can associate all requests together
		session := sessions.Default(ctx)
		session.Set("authFlowId", authFlow.Id)
		session.Save()

		// Kick off an auth we can have different kinds of auth
		if auth.OnAuthFlowStarted != nil {
			auth.OnAuthFlowStarted(authFlow, ctx)
		} else {
			// Not implemented
			ctx.JSON(501, gin.H{"error": "NotImplemented", "message": ""})
		}
	}
}

/**
 * Step 2. Called by the redirect/callback handler when credentials are provided
 * either by the user or by the auth provider.
 */
func (auth *Authenticator) HandleAuthFlowCredentials(ctx *gin.Context) {
	if auth.OnAuthFlowCredentialsReceived != nil {
		auth.OnAuthFlowCredentialsReceived(ctx)
	} else {
		// Not implemented
		ctx.JSON(501, gin.H{"error": "NotImplemented", "message": ""})
	}
}

/**
 * Step 3 - Method called auth has been verified with us receiving the verified
 * tokens etc.
 *
 * Here a channel is created along from the succeeded auth flow and is an
 * opportunity to extract identities and create users from this flow.
 *
 * This method is called after authFlowCredentialsReceived by the entity
 * verifying the auth flow credentials.
 */
func (auth *Authenticator) AuthFlowVerified(ctx *gin.Context, tokens map[string]interface{}, params map[string]interface{}, profile map[string]interface{}) (*Channel, *Identity) {
	// the ID here is specific to what is returned by the channel provider
	idFromProfile := auth.IdentityFromProfile(profile)
	// const authFlow = await datastore.getAuthFlowById(authFlowId);
	// TODO - Use the authFlow.purpose to ensure loginUser is not lost
	// ensure channel is created
	identity, _ := auth.IdentityStore.EnsureIdentity(idFromProfile.IdentityType, idFromProfile.IdentityKey, nil)
	channelParams := map[string]interface{}{
		"credentials": tokens,
		"profile":     profile,
		"identityKey": identity.Key(),
	}
	if _, ok := params["expires_in"]; ok {
		if val, ok := params["expires_in"].(int32); ok {
			channelParams["expires_in"] = val
		}
	}
	channel, _ := auth.ChannelStore.EnsureChannel(auth.Provider, idFromProfile.ChannelId, channelParams)
	if !channel.HasIdentity() {
		channel.IdentityKey = identity.Key()
		auth.ChannelStore.SaveChannel(channel)
	}

	if auth.OnIdentityEnsured != nil {
		auth.OnIdentityEnsured(channel, identity)
	}

	// Now ensure we also ensure an Identity entry here
	return channel, identity
}

/**
 * Step 4. After auth is successful this method is called to resume the
 * auth flow for the original purpose it was kicked off.
 */
func (auth *Authenticator) AuthFlowCompleted(ctx *gin.Context) {
	// Successful authentication, redirect success.
	q := ctx.Request.URL.Query()
	authFlowId := strings.Trim(q["state"][0], " ")
	authFlow := auth.AuthFlowStore.GetAuthFlowById(authFlowId)

	// We are done with the AuthFlow so clean it up
	session := sessions.Default(ctx)
	session.Set("authFlowId", nil)
	session.Save()

	if authFlow != nil && auth.ContinueAuthFlow != nil {
		auth.ContinueAuthFlow(authFlow, ctx)
	} else {
		ctx.Redirect(http.StatusFound, "/")
	}
}

/**
 * Ensures that an auth flow exists and that it matches that for this
 * given provider.
 */
func (auth *Authenticator) EnsureAuthFlow(authFlowId string, callbackURL string) (authFlow *AuthFlow) {
	authFlow = nil
	if authFlowId == "" {
		// create a new session if it was not provided
		authFlow = auth.AuthFlowStore.SaveAuthFlow(
			&AuthFlow{
				Provider:      auth.Provider,
				HandlerName:   "login",
				HandlerParams: map[string]interface{}{callbackURL: callbackURL},
			},
		)
	} else {
		authFlow = auth.AuthFlowStore.GetAuthFlowById(authFlowId)
	}
	if authFlow == nil || authFlow.Provider != auth.Provider {
		return nil
	}
	// session.authFlowId = authFlow.id;
	return
}
