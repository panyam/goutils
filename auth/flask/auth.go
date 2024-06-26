package flask

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"

	"encoding/json"
	"log"
	"strings"

	fernet "github.com/fernet/fernet-go"
	gut "github.com/panyam/goutils/utils"
)

type FlaskAuth struct {
	AppSecretKey string
	LogCookies   bool
}

func (f *FlaskAuth) NormalizedSecretKey() string {
	for len(f.AppSecretKey) < 32 {
		f.AppSecretKey += " "
	}
	if len(f.AppSecretKey) > 32 {
		f.AppSecretKey = f.AppSecretKey[:32]
	}
	return base64.StdEncoding.EncodeToString([]byte(f.AppSecretKey))
}

/**
 * Decodes the session cookie as it is stored by flask auth.
 * This has a few parts (and plugin points):
 */
func (f *FlaskAuth) DecodeSessionCookie(base64value string) (out gut.StrMap, err error) {
	decompress := base64value[0] == '.'
	if decompress {
		base64value = base64value[1:]
	}
	orig := base64value
	base64value = strings.Map(func(ch rune) rune {
		if ch == '-' {
			ch = '+'
		}
		if ch == '_' {
			ch = '/'
		}
		return ch
	}, base64value)
	parts := strings.Split(base64value, ".")
	var timestampBytes []byte
	if len(parts) >= 1 {
		if d, err := base64.StdEncoding.DecodeString(gut.PaddedWith(parts[1], '=')); err != nil {
			log.Println("Error decoding timestamp: ", err)
		} else {
			timestampBytes = d
			if f.LogCookies {
				log.Println("Decoded TimeStamp: ", timestampBytes)
			}
		}
	}
	var hmacBytes []byte
	if len(parts) >= 2 {
		if d, err := base64.StdEncoding.DecodeString(gut.PaddedWith(parts[2], '=')); err != nil {
			log.Println("Error decoding timestamp: ", err)
		} else {
			hmacBytes = d
			if f.LogCookies {
				log.Println("Decoded HMAC: ", hmacBytes)
			}
		}
	}
	base64_encoded_data := parts[0]
	padded := gut.PaddedWith(base64_encoded_data, '=')
	decoded, err := base64.StdEncoding.DecodeString(padded)
	if err != nil {
		log.Println("Error decoding: ", padded, err)
		log.Println("Orig Value: ", orig)
		return nil, err
	} else {
		if f.LogCookies {
			log.Println("Decoded Cookie: ", decoded)
		}
	}

	if decompress {
		if zr, err := zlib.NewReader(bytes.NewReader(decoded)); err != nil {
			log.Println("error decompressing decoded cookie: ", err)
			return nil, err
		} else if decoded, err = io.ReadAll(zr); err != nil {
			return nil, err
		}
	}

	if err = json.Unmarshal(decoded, &out); err != nil {
		log.Println("Error decoding json: ", padded, decoded, err)
		return nil, err
	}

	return
}

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

var ALPHABET_REVERSE = gut.AlphaReverseMap(ALPHABET, nil)

func (f *FlaskAuth) DecodeSessionUserId(userid string) (out []interface{}) {
	userid = gut.PaddedWith(userid, '=')
	// log.Printf("NormalizedKey: [%s]", f.NormalizedSecretKey())
	key := fernet.MustDecodeKeys(f.NormalizedSecretKey())
	data := fernet.VerifyAndDecrypt([]byte(userid), 0, key)
	// log.Println("Data: ", data)
	parts := strings.Split(string(data), "|")
	// log.Println("Data: ", userid, parts)
	if parts != nil {
		for _, part := range parts {
			if part[0] == '~' {
				out = append(out, gut.ExcelDecode(part[1:], ALPHABET, ALPHABET_REVERSE))
			} else {
				out = append(out, part)
			}
		}
	}
	return
}

func (f *FlaskAuth) ParseSignedCookieValue(value string) (parts []interface{}, sessmap gut.StrMap) {
	var err error
	sessmap, err = f.DecodeSessionCookie(value)
	if err != nil {
		log.Println("error processing session: ", err)
		return
	}
	user_id, ok := sessmap["_user_id"]
	if user_id == nil || !ok || user_id.(string) == "" {
		log.Println("could not find _user_id in cookie: ", err)
		return
	}

	parts = f.DecodeSessionUserId(user_id.(string))
	return
}
