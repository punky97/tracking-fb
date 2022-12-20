package transhttp

import (
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils/codec/bejson"
	"encoding/json"
	"github.com/spf13/viper"
	"net/http"
	"reflect"
	"strconv"

	"crypto/md5"
	"encoding/hex"
)

type PingResponse struct {
	Reply string `json:"reply"`
}

// RespondJSONFull - response all but no etag
func RespondJSONFull(w http.ResponseWriter, httpStatusCode int, payload interface{}) {
	RespondJSONFullWithETag(w, httpStatusCode, payload, "")
}

// GetMD5Hash - Get md5 tag
func GetMD5Hash(text []byte) string {
	hasher := md5.New()
	hasher.Write(text)
	return hex.EncodeToString(hasher.Sum(nil))
}

// RespondJSONFullWithETag - RespondJSON with force emit empty field -- makes the response with payload as json format
func RespondJSONFullWithETag(w http.ResponseWriter, httpStatusCode int, payload interface{}, ETag string) {
	data, err := bejson.Marshal(payload, true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// If enable etag
	if len(ETag) > 0 {
		// Add to header
		SetETag(w, ETag)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(httpStatusCode)
	w.Write(data)
}

// SetETag
func SetETag(w http.ResponseWriter, ETag string) {
	// Add to header
	w.Header().Set("ETag", "W/\""+ETag+"\"")
}

func ignoreOldAttribute(v interface{}) {
	initStatus := viper.GetBool("dirty_field.enable")
	if initStatus {
		nomalize(reflect.ValueOf(v))
	}
}

func nomalize(v reflect.Value) {
	if !v.IsValid() {
		logger.BkLog.Info("value is not valid")
		return
	}

	switch v.Kind() {
	case reflect.Ptr:
		nomalize(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			switch v.Field(i).Kind() {
			case reflect.Slice:
				if "UpdatedField" == v.Type().Field(i).Name {
					if v.Field(i).CanSet() {
						v.Field(i).Set(reflect.Zero(v.Field(i).Type()))
					}
				} else {
					for j := 0; j < v.Field(i).Len(); j++ {
						nomalize(v.Field(i).Index(j))
					}
				}
			case reflect.Map:
				if "OldAttribute" == v.Type().Field(i).Name {
					if v.Field(i).CanSet() {
						v.Field(i).Set(reflect.Zero(v.Field(i).Type()))
					}
				} else {
					for _, k := range v.Field(i).MapKeys() {
						nomalize(v.Field(i).MapIndex(k))
					}
				}
			}
		}
	default:
		break
	}
}

// RespondJSON -- makes the response with payload as json format
func RespondJSON(w http.ResponseWriter, httpStatusCode int, payload interface{}) {
	ignoreOldAttribute(payload)
	data, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(httpStatusCode)
	w.Write(data)
}

// RespondError -- makes the error response with payload as json format
func RespondError(w http.ResponseWriter, httpStatusCode int, message string) {
	RespondJSON(w, httpStatusCode, map[string]string{"error": message})
}

func RespondJSONError(w http.ResponseWriter, payload interface{}) {
	var httpStatusCode = http.StatusInternalServerError
	if err, ok := payload.(error); ok {
		httpStatusCode = GetStatusCode(err)
	}
	RespondJSON(w, httpStatusCode, payload)
}

// RespondMessage -- makes the message response with payload as json format
func RespondMessage(w http.ResponseWriter, httpStatusCode int, message string) {
	RespondJSON(w, httpStatusCode, map[string]string{"message": message})
}

// RespondMessageWithContentType -- makes the message response with payload as json format
func RespondMessageWithContentType(w http.ResponseWriter, httpStatusCode int, message string, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(message)))
	w.WriteHeader(httpStatusCode)
	w.Write([]byte(message))
}

// Redirect -- redirect
func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}

// RespondFile --
func RespondFile(w http.ResponseWriter, r *http.Request, fileName string) {
	http.ServeFile(w, r, fileName)
}
