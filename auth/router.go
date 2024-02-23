package auth

import (
	"fmt"
	"log"
	"net/http"

	gohttp "github.com/panyam/goutils/http"
	"github.com/panyam/goutils/utils"
)

type AuthConfig struct {
	RequestVars        *gohttp.RequestVarMap
	SessionGetter      func(r *http.Request, w http.ResponseWriter, param string) any
	CallbackURLParam   string
	DefaultRedirectURL string
	GetRedirURL        func(r *http.Request) string
	UserParamName      string
	NoLoginRedirect    bool
}

/**
 * Redirects users to login screen of they are not logged in
 * @param req Request object
 * @param res Response object
 * @param next next function
 */
func (a *AuthConfig) EnsureLogin(w http.ResponseWriter, r *http.Request) {
	if a.UserParamName == "" {
		a.UserParamName = "loggedInUserId"
	}
	if a.CallbackURLParam == "" {
		a.CallbackURLParam = "/callbackURL"
	}
	if a.DefaultRedirectURL == "" {
		a.DefaultRedirectURL = "/login"
	}
	if a.GetRedirURL == nil && !a.NoLoginRedirect {
		a.GetRedirURL = func(r *http.Request) string { return a.DefaultRedirectURL }
	}
	if a.UserParamName == "" {
		a.UserParamName = "loggedInUser"
	}
	userParam := a.SessionGetter(r, w, a.UserParamName)
	if userParam == "" || userParam == nil {
		// Redirect to a login if user not logged in
		// `/${a.redirectURLPrefix || "auth"}/login?callbackURL=${encodeURIComponent(req.originalUrl)}`;
		redirUrl := ""
		if a.GetRedirURL != nil {
			redirUrl = a.GetRedirURL(r)
		}
		if redirUrl != "" {
			originalUrl := r.URL.Path
			encodedUrl := utils.EncodeURIComponent(originalUrl)
			fullRedirUrl := fmt.Sprintf("%s?%s=%s", redirUrl, a.CallbackURLParam, encodedUrl)
			http.Redirect(w, r, fullRedirUrl, http.StatusFound)
		} else {
			// otherwise a 401
			http.Error(w, "Failed", http.StatusUnauthorized)
		}
	} else if a.RequestVars != nil {
		log.Println("Setting Logged In User Id: ", userParam)
		// ctx.Set("loggedInUserId", userParam)
		a.RequestVars.SetKey(r, "loggedInUser", userParam)
	}
}
