package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/panyam/goutils/dal"
	"github.com/panyam/goutils/utils"
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
	IdentityType string `gorm:"primaryKey"`

	// The key specific to the identity (eg an email address or a phone number etc).
	//
	// type + key should be unique through out the system.
	IdentityKey string `gorm:"primaryKey"`

	// The primary user that this identity can be associated with.
	// Identities do not need to be explicitly associted with a user especially
	// in systems where a single Identity can be used to front several users
	PrimaryUser string
}

func NewIdentity(idType string, idKey string) *Identity {
	out := Identity{
		IdentityType: idType,
		IdentityKey:  idKey,
		BaseEntity: dal.BaseEntity{
			CreatedAt: time.Now(),
		},
	}
	return &out
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

type AuthFlow struct {
	dal.BaseEntity

	// A unique Auth Session ID
	ID string

	// Kind of login being done
	Provider string

	// When this Auth session expires;
	ExpiresIn time.Time // 300

	// Call back URL for where the session needs to endup on success
	// callback: CallbackRequest;

	// Handler that will continue the flow after a successful AuthFlow.
	HandlerName string // "login"

	// Parameters for the handler to continue with.
	HandlerParams dal.JsonField `gorm:"type:text"`
}

// And others things here
func (af *AuthFlow) HasKey() bool {
	return strings.Trim(af.ID, " ") != ""
}

/**
 * Channel's represented federated verification objects.  For example a Google
 * Signin would ensure that the user that goes through this flow will end up
 * with a Google signin Channel - which would verify a particular identity type.
 */
type Channel struct {
	dal.BaseEntity

	Provider string `gorm:"primaryKey"`
	LoginId  string `gorm:"primaryKey"`

	/**
	 * Credentials for this channel (like access tokens, hashed passwords etc).
	 */
	Credentials dal.JsonField `gorm:"type:text"`

	/**
	 * Profile as passed by the provider of the channel.
	 */
	Profile dal.JsonField `gorm:"type:text"`

	/**
	 * When does this channel expire and needs another login/auth.
	 */
	ExpiresAt time.Time

	// The identity that this channel is verifying.
	IdentityKey string
}

func NewChannel(provider string, loginId string, params utils.StrMap) *Channel {
	out := Channel{
		Provider: provider,
		LoginId:  loginId,
		BaseEntity: dal.BaseEntity{
			CreatedAt: time.Now(),
		},
	}
	return &out
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
	Headers utils.StrMap
}

func (c *CallbackRequest) FullURL() string {
	if !strings.HasSuffix(c.Hostname, "/") && !strings.HasPrefix(c.Path, "/") {
		return c.Hostname + "/" + c.Path
	} else {
		return c.Hostname + c.Path
	}
}
