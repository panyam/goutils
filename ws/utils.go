package ws

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	gut "github.com/panyam/goutils/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func SendJsonResponse(writer http.ResponseWriter, resp interface{}, err error) {
	output := resp
	httpCode := ErrorToHttpCode(err)
	if err != nil {
		if er, ok := status.FromError(err); ok {
			code := er.Code()
			msg := er.Message()
			output = gut.StringMap{
				"error":   code,
				"message": msg,
			}
		} else {
			output = gut.StringMap{
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
