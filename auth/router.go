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
	SessionGetter      func(r *http.Request, param string) any
	CallbackURLParam   string
	DefaultRedirectURL string
	GetRedirURL        func(r *http.Request) string
	UserParamName      string
	NoLoginRedirect    bool
}

/**
 * Ensures that config values have reasonable defaults.
 */
func (a *AuthConfig) EnsureReasonableDefaults() {
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
}

func (a *AuthConfig) GetLoggedInUserId(r *http.Request) string {
	loggedInUserId := a.RequestVars.GetKey(r, "loggedInUser")
	log.Println("Getting Logged In User Id: ", loggedInUserId)
	if loggedInUserId == "" || loggedInUserId == nil {
		userParam := a.SessionGetter(r, a.UserParamName)
		if userParam != "" && userParam != nil {
			log.Println("Logged In User Id: ", userParam, loggedInUserId)
			return userParam.(string)
		} else {
			return ""
		}
	}
	return loggedInUserId.(string)
}

/**
 * Extracts user info from the request and saves it into current user.
 * Can be used by further middleware down the line to get the request's
 * user info
 */
func (a *AuthConfig) ExtractUserInfo(w http.ResponseWriter, r *http.Request) {
	userParam := a.SessionGetter(r, a.UserParamName)
	if userParam != "" && userParam != nil {
		a.RequestVars.SetKey(r, "loggedInUser", userParam)
	}
}

/**
 * Redirects users to login screen of they are not logged in
 * @param req Request object
 * @param res Response object
 * @param next next function
 */
func (a *AuthConfig) EnsureLoginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if a.EnsureLogin(w, r) {
				next.ServeHTTP(w, r)
			}
		},
	)
}

func (a *AuthConfig) EnsureLogin(w http.ResponseWriter, r *http.Request) bool {
	a.EnsureReasonableDefaults()
	userParam := a.SessionGetter(r, a.UserParamName)
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
			http.Error(w, "Login Failed", http.StatusUnauthorized)
		}
	} else if a.RequestVars != nil {
		log.Println("Setting Logged In User Id: ", userParam)
		// ctx.Set("loggedInUserId", userParam)
		a.RequestVars.SetKey(r, "loggedInUser", userParam)
		return true
	} else {
		http.Error(w, "ReqVarMap not set", http.StatusInternalServerError)
	}
	return false
}
