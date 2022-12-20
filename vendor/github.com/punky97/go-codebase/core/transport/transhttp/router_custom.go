package transhttp

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/punky97/go-codebase/core/libs/docgen/genswagger"
	file_parser "github.com/punky97/go-codebase/core/libs/docgen/swagger-file-parser"
	"net/http"
	"os"
	"reflect"
	"strings"
)

const SwaggerPath = "/swagger-doc"

type ExecuteRoute func(src1 interface{}, src2 interface{}) interface{}

type HandleIpl struct {
	execute          reflect.Type
	executeMeth      reflect.Value
	inputBodyType    reflect.Type
	queryParamType   reflect.Type
	responseBodyType reflect.Type

	notParseBody       bool
	notParseQueryParam bool
}

func (h HandleIpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var queryParam = reflect.New(h.queryParamType)
	var inputBody = reflect.New(h.inputBodyType)
	if h.inputBodyType != nil && !h.notParseBody {
		if !inputBody.IsNil() && r.Body != nil {
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(inputBody.Interface())
			if err != nil {
				RespondError(w, http.StatusBadRequest, "Request body wrong format: "+err.Error())
				return
			}
		}
	}
	if h.queryParamType != nil && !h.notParseQueryParam {
		err := schema.NewDecoder().Decode(queryParam.Interface(), r.URL.Query())
		if err != nil {
			RespondError(w, http.StatusBadRequest, "Request query wrong format: "+err.Error())
			return
		}
		queryParam = queryParam.Elem()
	} else {
		queryParam = reflect.ValueOf(r.URL.Query())
	}
	res := h.executeMeth.Call([]reflect.Value{queryParam, reflect.ValueOf(mux.Vars(r)),
		inputBody.Elem(), reflect.ValueOf(r.Context())})
	if !res[0].IsNil() {
		RespondJSON(w, http.StatusOK, res[0].Interface())
		return
	}
	if !res[1].IsNil() {
		RespondJSONError(w, res[1].Interface())
		return
	}
}

type SwaggerHandler struct {
	data interface{}
}

func (s SwaggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, s.data)
	return
}

// ConvertRoutes -- load all routers
func ConvertRoutes(wRoutes []Route, mainPath string, version string, namespace string) (Routes, interface{}) {
	swaggerInfo := genswagger.SwaggerServerInfo{}
	swaggerInfo.Version = version
	routes := make([]Route, 0)
	handlerMap := make(map[string]genswagger.SwaggerRouteInfo)
	for _, route := range wRoutes {
		if swaggerInfo.BasePath == "" && route.BasePath != "" {
			swaggerInfo.BasePath = route.BasePath
			swaggerInfo.Version, swaggerInfo.Name = parseBasePath(route.BasePath)
		}
		routeInfo := genswagger.SwaggerRouteInfo{
			Pattern:     route.Pattern,
			Method:      route.Method,
			Name:        route.Name,
			Description: route.Description,
			ResponseDes: route.ResponseDescription,
			RequestDes:  route.RequestDescription,
			Deprecated:  route.Deprecated,
		}
		if route.AuthInfo.TokenType != "" {
			routeInfo.AuthInfo = genswagger.AuthInfo{
				TokenType:      route.AuthInfo.TokenType,
				RequiredFields: route.AuthInfo.RequiredFields,
				RestrictScopes: route.AuthInfo.RestrictScopes,
			}
		}
		handlerMap[getHandler(route.Handler)] = routeInfo
	}
	if mainPath == "" {
		panic("mainPath is not set")
	}
	mainPath = strings.Replace(mainPath, os.Getenv("GOPATH")+"/src/", "", 1)
	mainAPIFile := "main.go"
	p := file_parser.New()
	p.HandlerName = handlerMap
	p.SwaggerServerInfo = swaggerInfo
	p.Namespace = namespace
	data, err := p.ParseAPI(mainPath, mainAPIFile)
	if err != nil {
		panic("Error when ParseAPI, err: " + err.Error())
	}
	return routes, data
}

func parseBasePath(basePath string) (version, name string) {
	pathSplit := strings.Split(basePath, "/")
	if len(pathSplit) < 3 {
		return "", basePath
	}
	return pathSplit[1], pathSplit[2]
}

// NewRouter -- load all routers
func getHandler(handler http.Handler) string {
	handlerName := reflect.TypeOf(handler).String()
	handlerNameSp := strings.Split(handlerName, ".")
	return handlerNameSp[len(handlerNameSp)-1]
}
