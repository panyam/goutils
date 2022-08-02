package auth

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/panyam/goutils/utils"
	"net/http"
	"net/url"
	"strings"
)

type EnsureLoginConfig struct {
	GetRedirURL   func(ctx *gin.Context) string
	UserParamName string
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
	if config.GetRedirURL == nil {
		config.GetRedirURL = func(ctx *gin.Context) string { return "/auth/login" }
	}
	if config.UserParamName == "" {
		config.UserParamName = "loggedInUser"
	}
	return func(ctx *gin.Context) {
		session := sessions.Default(ctx)
		userParam := session.Get(config.UserParamName)
		if userParam == "" || userParam == nil {
			// Redirect to a login if user not logged in
			// `/${config.redirectURLPrefix || "auth"}/login?callbackURL=${encodeURIComponent(req.originalUrl)}`;
			redirUrl := config.GetRedirURL(ctx)
			originalUrl := ctx.Request.URL.Path
			encodedUrl := utils.EncodeURIComponent(originalUrl)
			fullRedirUrl := fmt.Sprintf("%s?callbackURL=%s", redirUrl, encodedUrl)
			ctx.Redirect(http.StatusFound, fullRedirUrl)
		}
	}
}
