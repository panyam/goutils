package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/panyam/goutils/conc"
	gut "github.com/panyam/goutils/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func JsonToQueryString(json map[string]any) string {
	out := ""
	for key, value := range json {
		if len(out) > 0 {
			out += "&"
		}
		item := fmt.Sprintf("%s=%v", url.PathEscape(key), value.(interface{}))
		out += item
	}
	return out
}

func SendJsonResponse(writer http.ResponseWriter, resp interface{}, err error) {
	output := resp
	httpCode := ErrorToHttpCode(err)
	if err != nil {
		if er, ok := status.FromError(err); ok {
			code := er.Code()
			msg := er.Message()
			output = gut.StrMap{
				"error":   code,
				"message": msg,
			}
		} else {
			output = gut.StrMap{
				"error": err.Error(),
			}
		}
	}
	writer.WriteHeader(httpCode)
	writer.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(output)
	if err != nil {
		log.Println("Error happened in JSON marshal. Err: ", err)
	}
	writer.Write(jsonResp)
}

func ErrorToHttpCode(err error) int {
	httpCode := http.StatusOK
	if err != nil {
		httpCode = http.StatusInternalServerError
		if er, ok := status.FromError(err); ok {
			code := er.Code()
			// msg := er.Message()
			// see if we have a specific client error
			if code == codes.PermissionDenied {
				httpCode = http.StatusForbidden
			} else if code == codes.NotFound {
				httpCode = http.StatusNotFound
			} else if code == codes.AlreadyExists {
				httpCode = http.StatusConflict
			} else if code == codes.InvalidArgument {
				httpCode = http.StatusBadRequest
			}
		}
	}
	return httpCode
}

func WSConnWriteError(wsConn *websocket.Conn, err error) error {
	if err != nil && err != io.EOF {
		// Some kind of streamer rpc error
		log.Println("Error reading message from streamer: ", err)
		errdata := make(map[string]interface{})
		if er, ok := status.FromError(err); ok {
			errdata["error"] = er.Code()
		}
		jsonData, outerr := json.Marshal(errdata)
		if outerr != nil {
			outerr = wsConn.WriteMessage(websocket.TextMessage, jsonData)
		}
		if outerr != nil {
			log.Println("Error sending message: ", err)
		}
		return outerr
	}
	return nil
}

func WSConnWriteMessage(wsConn *websocket.Conn, msg interface{}) error {
	jsonResp, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error happened in JSON marshal. Err: ", err)
	}
	outerr := wsConn.WriteMessage(websocket.TextMessage, jsonResp)
	if err != nil {
		log.Println("Error sending message: ", err)
	}
	return outerr
}

func WSConnJSONReaderWriter(conn *websocket.Conn) (reader *conc.Reader[gut.StrMap], writer *conc.Writer[conc.Message[gut.StrMap]]) {
	reader = conc.NewReader(func() (out gut.StrMap, err error) {
		err = conn.ReadJSON(&out)
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
			err = net.ErrClosed
		}
		return
	})
	writer = conc.NewWriter(func(msg conc.Message[gut.StrMap]) error {
		if msg.Error == io.EOF {
			log.Println("Streamer closed...", msg.Error)
			// do nothing
			// SendJsonResponse(rw, nil, msg.Error)
			return msg.Error
		} else if msg.Error != nil {
			return WSConnWriteError(conn, msg.Error)
		} else {
			return WSConnWriteMessage(conn, msg.Value)
		}
	})
	return
}

func NormalizeWsUrl(httpOrWsUrl string) string {
	if strings.HasSuffix(httpOrWsUrl, "/") {
		httpOrWsUrl = (httpOrWsUrl)[:len(httpOrWsUrl)-1]
	}
	if strings.HasPrefix(httpOrWsUrl, "http:") {
		httpOrWsUrl = "ws:" + (httpOrWsUrl)[len("http:"):]
	}
	if strings.HasPrefix(httpOrWsUrl, "https:") {
		httpOrWsUrl = "wss:" + (httpOrWsUrl)[len("https:"):]
	}
	return httpOrWsUrl
}
