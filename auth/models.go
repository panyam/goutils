package auth

import (
	"fmt"
	"github.com/panyam/goutils/dal"
	"strings"
	"time"
)

/**
 * An identify is a unique global "address" corresponding to a user.
 * For example the identify abc@example.com is a unique identify regardless
 * of which Channel is verifying it.  Multiple channels can verify the same
 * entity, eg open auth by github, FB or Google can verify the same email
 * address.
 */
type Identity struct {
	dal.BaseEntity

	// Type of identity being verified (eg email, phone etc).
	IdentityType string

	// The key specific to the identity (eg an email address or a phone number etc).
	//
	// type + key should be unique through out the system.
	IdentityKey string

	// The primary user that this identity can be associated with.
	// Identities do not need to be explicitly associted with a user especially
	// in systems where a single Identity can be used to front several users
	PrimaryUser string
}

func (id *Identity) HasUser() bool {
	return strings.Trim(id.PrimaryUser, " ") != ""
}

func (id *Identity) HasKey() bool {
	return strings.Trim(id.IdentityType, " ") != "" &&
		strings.Trim(id.IdentityKey, " ") != ""
}

func (id *Identity) Key() string {
	return fmt.Sprintf("%s:%s", id.IdentityType, id.IdentityKey)
}

/**
 * Once a channel has verified an Identity, the end result is a mapping to
 * a local user object that is the entry for authenticated actions within
 * the system.  The User can also mean a user profile and can be extended
 * to be customized by the user of this library in their own specific app.
 */
type User struct {
	dal.BaseEntity

	// A globally unique user ID.  This User ID cannot be used as a login key.
	// Login's need to happen via the Identiites above and a username could be
	// one of the identities (which can be verified say via login/password mechanism)
	// Alternatively an email can be used as an identity that can also map to
	// a particular user.
	Id string

	ProfileData map[string]interface{}
}

// And others things here
func (user *User) HasKey() bool {
	return strings.Trim(user.Id, " ") != ""
}

type AuthFlow struct {
	dal.BaseEntity

	// A unique Auth Session ID
	Id string

	// Kind of login being done
	Provider string

	// When this Auth session expires;
	ExpiresIn time.Time // 300

	// Call back URL for where the session needs to endup on success
	// callback: CallbackRequest;

	// Handler that will continue the flow after a successful AuthFlow.
	HandlerName string // "login"

	// Parameters for the handler to continue with.
	HandlerParams map[string]interface{}
}

// And others things here
func (af *AuthFlow) HasKey() bool {
	return strings.Trim(af.Id, " ") != ""
}

/**
 * Channel's represented federated verification objects.  For example a Google
 * Signin would ensure that the user that goes through this flow will end up with
 * a Google signin Channel - which would verify a particular identity type.
 */
type Channel struct {
	dal.BaseEntity

	Provider string
	LoginId  string

	/**
	 * Credentials for this channel (like access tokens, passwords etc).
	 */
	Credentials map[string]interface{}

	/**
	 * Profile as passed by the provider of the channel.
	 */
	Profile map[string]interface{}

	/**
	 * When does this channel expire and needs another login/auth.
	 */
	ExpiresIn time.Time

	// The identity that this channel is verifying.
	IdentityKey string
}

func (ch *Channel) HasIdentity() bool {
	return strings.Trim(ch.IdentityKey, " ") != ""
}

func (ch *Channel) Key() string {
	return fmt.Sprintf("%s:%s", ch.Provider, ch.LoginId)
}

func (ch *Channel) HasKey() bool {
	return strings.Trim(ch.Provider, " ") != "" &&
		strings.Trim(ch.LoginId, " ") != ""
}

type CallbackRequest struct {
	Hostname string

	Path string

	// Method to call the callback URL on
	Method string

	// For POST/PUT methods
	rawBody string

	// Headers for this request
	Headers map[string]interface{}
}

/*
  constructor(config: any) {
    config = config || {};
    this.hostname = config.hostname;
    this.path = config.path || "/";
    if (config.url && config.url.length > 0) {
      // parse hostnamea and path from url
      if (config.url.indexOf("://") < 0) {
        throw new Error("Invalid URL: " + config.url + ".  Please use hostname and path instead.");
      }
      const u = new URL(config.url);
      console.log("Parsed URL: ", u);
    }
    this.method = config.method || "GET";
    this.headers = config.headers || {};
    this.rawBody = config.rawBody || "";
  }
*/

func (c *CallbackRequest) FullURL() string {
	if !strings.HasSuffix(c.Hostname, "/") && !strings.HasPrefix(c.Path, "/") {
		return c.Hostname + "/" + c.Path
	} else {
		return c.Hostname + c.Path
	}
}
