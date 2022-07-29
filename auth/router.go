package auth

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
)

/**
 * Registers endpoints required for OAuth under a given router.
 */
func RegisterOAuthRouter(
	rg *gin.RouterGroup,
	authn *Authenticator,
	startPrefix string, ///  = "/",
	successCallbackPrefix string, ////  = "/callback",
	failureCallbackPrefix string, ///  = "/fail",
) {
	/*
	 * Registers a starting endpoint for the auth.
	 *
	 * This is usually a get request that kicks off the auth.  This can also
	 * be overridden to handle other reuqests (eg post requests).
	 */
	rg.GET(startPrefix, func(ctx *gin.Context) {
		authn.StartAuthFlow(ctx)
	})

	/**
	 * Registers an endpoint for handling any processing of credentials either by the
	 * user (eg username/password) or by auth providers (via success callbacks).
	 */
	rg.GET(successCallbackPrefix, authn.HandleAuthFlowCredentials, authn.AuthFlowSucceeded)

	/**
	 * Registers the failure endpoint handler.
	 */
	rg.GET(failureCallbackPrefix, authn.AuthFlowFailed)
}

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
func (auth *Authenticator) EnsureLogin(config *EnsureLoginConfig) RequestHandler {
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
		}
	}
}
