package auth

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/panyam/goutils/utils"
)

type RequestHandler = func(ctx *gin.Context)

type EnsureLoginConfig struct {
	CallbackURLParam   string
	DefaultRedirectURL string
	GetRedirURL        func(ctx *gin.Context) string
	SessionGetter      func(ctx *gin.Context, param string) any
	UserParamName      string
	NoLoginRedirect    bool
}

/**
 * Redirects users to login screen of they are not logged in
 * @param req Request object
 * @param res Response object
 * @param next next function
 */
func EnsureLogin(config *EnsureLoginConfig) RequestHandler {
	if config == nil {
		config = &EnsureLoginConfig{}
	}
	if config.UserParamName == "" {
		config.UserParamName = "loggedInUserId"
	}
	if config.CallbackURLParam == "" {
		config.CallbackURLParam = "/callbackURL"
	}
	if config.DefaultRedirectURL == "" {
		config.DefaultRedirectURL = "/login"
	}
	if config.GetRedirURL == nil && !config.NoLoginRedirect {
		config.GetRedirURL = func(ctx *gin.Context) string { return config.DefaultRedirectURL }
	}
	if config.UserParamName == "" {
		config.UserParamName = "loggedInUser"
	}
	if config.SessionGetter == nil {
		config.SessionGetter = func(ctx *gin.Context, param string) any {
			session := sessions.Default(ctx)
			return session.Get(param)
		}
	}
	return func(ctx *gin.Context) {
		userParam := config.SessionGetter(ctx, config.UserParamName)
		if userParam == "" || userParam == nil {
			// Redirect to a login if user not logged in
			// `/${config.redirectURLPrefix || "auth"}/login?callbackURL=${encodeURIComponent(req.originalUrl)}`;
			redirUrl := ""
			if config.GetRedirURL != nil {
				redirUrl = config.GetRedirURL(ctx)
			}
			if redirUrl != "" {
				originalUrl := ctx.Request.URL.Path
				encodedUrl := utils.EncodeURIComponent(originalUrl)
				fullRedirUrl := fmt.Sprintf("%s?%s=%s", redirUrl, config.CallbackURLParam, encodedUrl)
				ctx.Redirect(http.StatusFound, fullRedirUrl)
			} else {
				// otherwise a 401
				ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Not LoggedIn"})
			}
			ctx.Abort()
		} else {
			log.Println("Setting Logged In User Id: ", userParam)
			ctx.Set("loggedInUserId", userParam)
		}
	}
}
