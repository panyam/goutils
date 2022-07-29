package auth

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
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
type AuthFlowSucceededFunc = func(authFlow *AuthFlow, ctx *gin.Context)
type AuthFlowFailedFunc = func(authFlow *AuthFlow, ctx *gin.Context)
type AuthFlowIdentityEnsured = func(channel *Channel, identity *Identity)

/**
 * A class that handles the multi step auth flow for verifying (and identifying) an
 * identity.
 *
 * The general workflow is as follows:
 *
 * 1. A user tries to access a resource at url ResourceUrl
 * 2. The request handler for ResourceUrl decides user is either not authenticated or
 *		not authorized
 *    - A user can be logged in but may require a different Agent or Role to be handy
 *			to authorize access.
 *    - The Authorizer will be redirected to at this point - getAuthorizer(ResourceUrl).startAuth(this);
 * 3. The Authorizer here may perform the auth without any control of "us".
 * 4. If Auth fails, the Authorizer will call back our failure endpoint.
 * 5. If AUth succeeds, the Authorizer will call our callback endpoint;
 *
 * General flow goes as follows:
 *
 * 1. User Visits  /xyz
 * 2. Request handler for /xyz kicks off an auth if not logged in (more in step 10).
 * 3. Handler(/xyz) redirects to /auth/login?callbackURL=<callbackURL>
 * 4. User selects one of the login types /auth/<provider>/login?callbackURL=<callbackURL>
 * 5. Login handler creates a new authFlow instance and saves it in the
 *    session.  This will be used later.  In (3) instead of a callbackURL
 *    an authFlow can also be provided in which a new AuthFlow wont be
 *    created.
 * 6. Here passport forwards off to the provider login callback to
 *    perform all manner of logins.
 * 7. After login is completed the callback URL is called with credentials
 *    (or failures in which case failure redirect is called).
 * 8. Here continueAuthFlow is called with a successful auth flow.
 * 9. Here we have the chance to handle the channel that was created
 *      - saved as req.currChannel
 * 10. Now is a chance to create req.loggedInUser so it is available for
 *     other requests going forward.
 */
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
	OnAuthFlowSucceeded AuthFlowSucceededFunc
	OnAuthFlowFailed    AuthFlowFailedFunc
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
		session.Set("authFlowId", authFlow.ID)
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
func (auth *Authenticator) AuthFlowSucceeded(ctx *gin.Context) {
	// Successful authentication, redirect success.
	auth.AuthFlowCompleted(true, ctx)
}

/**
 * Step 4b. Instead of step 4, this is called (by the provider)
 * if auth failed.
 */
func (auth *Authenticator) AuthFlowFailed(ctx *gin.Context) {
	auth.AuthFlowCompleted(false, ctx)
}

func (auth *Authenticator) AuthFlowCompleted(success bool, ctx *gin.Context) {
	q := ctx.Request.URL.Query()
	authFlowId := strings.Trim(q["state"][0], " ")
	authFlow, err := auth.AuthFlowStore.GetAuthFlowById(authFlowId)
	if err != nil {
		log.Println("Error getting auth flow id: ", authFlowId, err)
	}

	// We are done with the AuthFlow so clean it up
	session := sessions.Default(ctx)
	session.Set("authFlowId", nil)
	session.Save()

	if success {
		if authFlow != nil && auth.OnAuthFlowSucceeded != nil {
			auth.OnAuthFlowSucceeded(authFlow, ctx)
		} else {
			ctx.Redirect(http.StatusFound, "/")
		}
	} else {
		if auth.OnAuthFlowFailed != nil {
			auth.OnAuthFlowFailed(authFlow, ctx)
		} else {
			ctx.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": fmt.Sprintf("Login Failed for Provider: %s", auth.Provider),
			})
		}
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
		var err error
		authFlow, err = auth.AuthFlowStore.GetAuthFlowById(authFlowId)
		if err != nil {
			log.Println("Error getting authflow: ", authFlowId, err)
		}
	}
	if authFlow == nil || authFlow.Provider != auth.Provider {
		return nil
	}
	// session.authFlowId = authFlow.id;
	return
}
