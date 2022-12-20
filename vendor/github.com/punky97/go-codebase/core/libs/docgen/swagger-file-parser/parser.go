package swag

import (
	"github.com/punky97/go-codebase/core/libs/docgen/genswagger"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

const (
	// CamelCase indicates using CamelCase strategy for struct field.
	CamelCase = "camelcase"

	// PascalCase indicates using PascalCase strategy for struct field.
	PascalCase = "pascalcase"

	// SnakeCase indicates using SnakeCase strategy for struct field.
	SnakeCase = "snakecase"
)

type parseCodenameInfo struct {
	name     string
	basePath string
	tags     []string
}

type handlerUse struct {
	// sort all import
	imports []ast.ImportSpec
	//
	handlers map[string]string
	//
	HandlerName map[string]genswagger.SwaggerRouteInfo
}

// Parser implements a parser for Go source files.
type Parser struct {
	// basePath is $($BKGO)/src/
	basePath string

	//HandlerUse map[string]handlerUse

	// HandlerName store [handler struct name ][route info]
	HandlerName map[string]genswagger.SwaggerRouteInfo

	// HandlerName store [var code name][parseCodenameInfo] exp: map{"APIAuth": {Codename: "auth", HTTPBasePath: "/v1/auth"}}
	CodeName map[string]parseCodenameInfo

	SwaggerServerInfo genswagger.SwaggerServerInfo

	// swagger represents the root document object for the API specification
	swagger *genswagger.SwaggerObject

	//files is a map that stores map[real_go_file_path][astFile]
	files map[string]*ast.File

	// store path to
	filePathPkgStore map[string]map[string]string

	//files is a map that stores map[codename]map[real_go_file_path][astFile]
	multiFiles map[*parseCodenameInfo]map[string]*ast.File

	// in case we can scan multiple api service project
	currentCodename *parseCodenameInfo

	// fullPathDefinitions is a map that stores [full path][type name][*ast.TypeSpec]
	fullPathDefinitions FullPathDefinitions

	// schemaInFile is map stores [schema name][file path]
	schemaInFile SchemaInFile

	// CustomPrimitiveTypes is a map that stores custom primitive types to actual golang types [type name][string]
	CustomPrimitiveTypes map[string]string

	//registerTypes is a map that stores [refTypeName][*ast.TypeSpec]
	registerTypes map[string]*ast.TypeSpec

	PropNamingStrategy string

	// structStack stores full names of the structures that were already parsed or are being parsed now
	structStack []string

	// pathParsed check this path (exp: /bkgo/beekit/libs/docgen) this parse all file, [path][is parsed]
	pathParsed map[string]bool
	// check schema name was parsed (exp: /bkgo/beekit/libs/docgen/file-parser/pkg-test/models/core.Contact), [schema full name path][is parsed]
	schemaParsedDefinitions map[string]bool

	// is api route define schema name type is core_models.Order but full path of this is 'bkgo/exmsg/core/models.Order', [schema sort name][full path name]
	schemaSortName2FullPath UrlSchemaFullPath
	// Definitions temporary this is store full path of schema type so this should be clean again
	Definitions genswagger.Definitions `json:"definitions"`
	// pkgFileInfo store all file of pkg path
	pkgFileInfo PkgFileInfo
	// levelName same level clean of full path. exp: bkgo/exmsg/core/models.Order
	// level 0: Order
	// level 1: models.Order
	// level 2: core_models.Order
	// level 3: bkgo_exmsg_core_models.Order
	levelName map[string]int
	// count number of use of name
	countUseName map[string]int
	// trueNameDefinition same full path name and name should use (bkgo/exmsg/core/models.Order -> models.Order)
	trueNameDefinition        map[string]string
	trueNameDefinitionReverse map[string][]string

	// namespace: `internal` `external` `beta`...
	Namespace string
}

type StructPathStack map[string]map[string]bool
type FullPathDefinitions map[string]map[string]*ast.TypeSpec
type SchemaInFile map[string]map[string]*ast.File
type UrlSchemaFullPath map[string]map[string]string
type PkgFileInfo map[string][]os.FileInfo

var authDefine = map[string]*genswagger.SwaggerSecuritySchemeObject{
	"USER_ACCESS_TOKEN": {
		Type:        "apiKey",
		Description: "",
		Name:        "USER_ACCESS_TOKEN",
		In:          "header",
	},
	"APP_ACCESS_TOKEN": {
		Type:        "apiKey",
		Description: "",
		Name:        "APP_ACCESS_TOKEN",
		In:          "header",
	},
	"RESET_PASSWORD_TOKEN": {
		Type:        "apiKey",
		Description: "",
		Name:        "RESET_PASSWORD_TOKEN",
		In:          "header",
	},
	"CHANGE_EMAIL_ACCOUNT_TOKEN": {
		Type:        "apiKey",
		Description: "",
		Name:        "CHANGE_EMAIL_ACCOUNT_TOKEN",
		In:          "header",
	},
	"AUTHORIZE_CODE": {
		Type:        "apiKey",
		Description: "",
		Name:        "AUTHORIZE_CODE",
		In:          "header",
	},
	"SHOP_API_KEY": {
		Type:        "apiKey",
		Description: "",
		Name:        "SHOP_API_KEY",
		In:          "header",
	},
	"SIGN_UP_TOKEN": {
		Type:        "apiKey",
		Description: "",
		Name:        "SIGN_UP_TOKEN",
		In:          "header",
	},
	"SHOP_ACCESS_TOKEN": {
		Type:        "apiKey",
		Description: "",
		Name:        "SHOP_ACCESS_TOKEN",
		In:          "header",
	},
	"PRIVATE_APP_SHOP_ACCESS_TOKEN": {
		Type:        "apiKey",
		Description: "",
		Name:        "PRIVATE_APP_SHOP_ACCESS_TOKEN",
		In:          "header",
	},
}

// New creates a new Parser with default properties.
func New() *Parser {
	parser := &Parser{
		swagger: &genswagger.SwaggerObject{
			Info: genswagger.SwaggerInfoObject{
				Contact: &genswagger.SwaggerContactObject{},
				License: &genswagger.SwaggerLicenseObject{},
			},
			Paths:       make(map[string]genswagger.SwaggerPathItemObject),
			Definitions: make(map[string]*genswagger.Type),
		},
		files:                     make(map[string]*ast.File),
		CustomPrimitiveTypes:      make(map[string]string),
		registerTypes:             make(map[string]*ast.TypeSpec),
		pathParsed:                make(map[string]bool),
		schemaSortName2FullPath:   make(UrlSchemaFullPath),
		fullPathDefinitions:       make(FullPathDefinitions),
		schemaInFile:              make(SchemaInFile),
		basePath:                  os.Getenv("GOPATH") + "/src/",
		schemaParsedDefinitions:   make(map[string]bool),
		Definitions:               make(map[string]*genswagger.Type),
		pkgFileInfo:               make(PkgFileInfo),
		levelName:                 make(map[string]int),
		countUseName:              make(map[string]int),
		trueNameDefinition:        make(map[string]string),
		trueNameDefinitionReverse: make(map[string][]string),
		HandlerName:               map[string]genswagger.SwaggerRouteInfo{},
		CodeName:                  map[string]parseCodenameInfo{},
		multiFiles:                map[*parseCodenameInfo]map[string]*ast.File{},
		filePathPkgStore:          map[string]map[string]string{},
	}
	return parser
}

// ParseGeneralAPIInfo parses general api info for gived mainAPIFile path
func (parser *Parser) ParseGeneralAPIInfo(mainAPIFile string) error {
	fileSet := token.NewFileSet()
	fileTree, err := goparser.ParseFile(fileSet, mainAPIFile, nil, goparser.ParseComments)
	if err != nil {
		return errors.Wrap(err, "cannot parse soure files")
	}

	parser.swagger.Swagger = "2.0"
	securityMap := map[string]*genswagger.SwaggerSecuritySchemeObject{}

	// templated defaults
	if parser.SwaggerServerInfo.Version != "" {
		parser.swagger.Info.Version = parser.SwaggerServerInfo.Version
	} else {
		parser.swagger.Info.Version = "{{.Version}}"
	}
	if parser.SwaggerServerInfo.Name != "" {
		parser.swagger.Info.Title = parser.SwaggerServerInfo.Name
	} else {
		parser.swagger.Info.Title = "{{.Title}}"
	}
	parser.swagger.Info.Description = "{{.Description}}"
	parser.swagger.Host = "{{.Host}}"
	if parser.SwaggerServerInfo.BasePath != "" {
		parser.swagger.BasePath = parser.SwaggerServerInfo.BasePath
	} else {
		parser.swagger.BasePath = "{{.BasePath}}"
	}

	if fileTree.Comments != nil {
		for _, comment := range fileTree.Comments {
			comments := strings.Split(comment.Text(), "\n")
			for _, commentLine := range comments {
				attribute := strings.ToLower(strings.Split(commentLine, " ")[0])
				switch attribute {
				case "@version":
					parser.swagger.Info.Version = strings.TrimSpace(commentLine[len(attribute):])
				case "@title":
					parser.swagger.Info.Title = strings.TrimSpace(commentLine[len(attribute):])
				case "@description":
					if parser.swagger.Info.Description == "{{.Description}}" {
						parser.swagger.Info.Description = strings.TrimSpace(commentLine[len(attribute):])
					} else {
						parser.swagger.Info.Description += "\n" + strings.TrimSpace(commentLine[len(attribute):])
					}
				case "@termsofservice":
					parser.swagger.Info.TermsOfService = strings.TrimSpace(commentLine[len(attribute):])
				case "@contact.name":
					parser.swagger.Info.Contact.Name = strings.TrimSpace(commentLine[len(attribute):])
				case "@contact.email":
					parser.swagger.Info.Contact.Email = strings.TrimSpace(commentLine[len(attribute):])
				case "@contact.url":
					parser.swagger.Info.Contact.URL = strings.TrimSpace(commentLine[len(attribute):])
				case "@license.name":
					parser.swagger.Info.License.Name = strings.TrimSpace(commentLine[len(attribute):])
				case "@license.url":
					parser.swagger.Info.License.URL = strings.TrimSpace(commentLine[len(attribute):])
				case "@host":
					parser.swagger.Host = strings.TrimSpace(commentLine[len(attribute):])
				case "@basepath":
					parser.swagger.BasePath = strings.TrimSpace(commentLine[len(attribute):])
				case "@schemes":
					parser.swagger.Schemes = GetSchemes(commentLine)
				case "@tag.name":
					commentInfo := strings.TrimSpace(commentLine[len(attribute):])
					parser.swagger.Tags = append(parser.swagger.Tags, genswagger.SwaggerTag{
						Name: strings.TrimSpace(commentInfo),
					})
				case "@tag.description":
					commentInfo := strings.TrimSpace(commentLine[len(attribute):])
					tag := parser.swagger.Tags[len(parser.swagger.Tags)-1]
					tag.Description = commentInfo
					replaceLastTag(parser.swagger.Tags, tag)
				case "@tag.docs.url":
					commentInfo := strings.TrimSpace(commentLine[len(attribute):])
					tag := parser.swagger.Tags[len(parser.swagger.Tags)-1]
					tag.ExternalDocs = &genswagger.ExternalDocumentation{
						URL: commentInfo,
					}
					replaceLastTag(parser.swagger.Tags, tag)

				case "@tag.docs.description":
					commentInfo := strings.TrimSpace(commentLine[len(attribute):])
					tag := parser.swagger.Tags[len(parser.swagger.Tags)-1]
					if tag.ExternalDocs == nil {
						return errors.New("@tag.docs.description needs to come after a @tags.docs.url")
					}
					tag.ExternalDocs.Description = commentInfo
					replaceLastTag(parser.swagger.Tags, tag)
				}
			}

			for i := 0; i < len(comments); i++ {
				attribute := strings.ToLower(strings.Split(comments[i], " ")[0])
				switch attribute {
				case "@securitydefinitions.basic":
					securityMap[strings.TrimSpace(comments[i][len(attribute):])] = genswagger.BasicAuth()
				case "@securitydefinitions.apikey":
					attrMap := map[string]string{}
					for _, v := range comments[i+1:] {
						securityAttr := strings.ToLower(strings.Split(v, " ")[0])
						if securityAttr == "@in" || securityAttr == "@name" {
							attrMap[securityAttr] = strings.TrimSpace(v[len(securityAttr):])
						}
						// next securityDefinitions
						if strings.Index(securityAttr, "@securitydefinitions.") == 0 {
							break
						}
					}
					if len(attrMap) != 2 {
						return errors.New("@securitydefinitions.apikey is @name and @in required")
					}
					securityMap[strings.TrimSpace(comments[i][len(attribute):])] = genswagger.APIKeyAuth(attrMap["@name"], attrMap["@in"])
				case "@securitydefinitions.oauth2.application":
					attrMap := map[string]string{}
					scopes := map[string]string{}
					for _, v := range comments[i+1:] {
						securityAttr := strings.ToLower(strings.Split(v, " ")[0])
						if securityAttr == "@tokenurl" {
							attrMap[securityAttr] = strings.TrimSpace(v[len(securityAttr):])
						} else if isExistsScope(securityAttr) {
							scopes[getScopeScheme(securityAttr)] = v[len(securityAttr):]
						}
						// next securityDefinitions
						if strings.Index(securityAttr, "@securitydefinitions.") == 0 {
							break
						}
					}
					if len(attrMap) != 1 {
						return errors.New("@securitydefinitions.oauth2.application is @tokenUrl required")
					}
					securityScheme := genswagger.OAuth2Application(attrMap["@tokenurl"])
					for scope, description := range scopes {
						securityScheme.AddScope(scope, description)
					}
					securityMap[strings.TrimSpace(comments[i][len(attribute):])] = securityScheme
				case "@securitydefinitions.oauth2.implicit":
					attrMap := map[string]string{}
					scopes := map[string]string{}
					for _, v := range comments[i+1:] {
						securityAttr := strings.ToLower(strings.Split(v, " ")[0])
						if securityAttr == "@authorizationurl" {
							attrMap[securityAttr] = strings.TrimSpace(v[len(securityAttr):])
						} else if isExistsScope(securityAttr) {
							scopes[getScopeScheme(securityAttr)] = v[len(securityAttr):]
						}
						// next securityDefinitions
						if strings.Index(securityAttr, "@securitydefinitions.") == 0 {
							break
						}
					}
					if len(attrMap) != 1 {
						return errors.New("@securitydefinitions.oauth2.implicit is @authorizationUrl required")
					}
					securityScheme := genswagger.OAuth2Implicit(attrMap["@authorizationurl"])
					for scope, description := range scopes {
						securityScheme.AddScope(scope, description)
					}
					securityMap[strings.TrimSpace(comments[i][len(attribute):])] = securityScheme
				case "@securitydefinitions.oauth2.password":
					attrMap := map[string]string{}
					scopes := map[string]string{}
					for _, v := range comments[i+1:] {
						securityAttr := strings.ToLower(strings.Split(v, " ")[0])
						if securityAttr == "@tokenurl" {
							attrMap[securityAttr] = strings.TrimSpace(v[len(securityAttr):])
						} else if isExistsScope(securityAttr) {
							scopes[getScopeScheme(securityAttr)] = v[len(securityAttr):]
						}
						// next securityDefinitions
						if strings.Index(securityAttr, "@securitydefinitions.") == 0 {
							break
						}
					}
					if len(attrMap) != 1 {
						return errors.New("@securitydefinitions.oauth2.password is @tokenUrl required")
					}
					securityScheme := genswagger.OAuth2Password(attrMap["@tokenurl"])
					for scope, description := range scopes {
						securityScheme.AddScope(scope, description)
					}
					securityMap[strings.TrimSpace(comments[i][len(attribute):])] = securityScheme
				case "@securitydefinitions.oauth2.accesscode":
					attrMap := map[string]string{}
					scopes := map[string]string{}
					for _, v := range comments[i+1:] {
						securityAttr := strings.ToLower(strings.Split(v, " ")[0])
						if securityAttr == "@tokenurl" || securityAttr == "@authorizationurl" {
							attrMap[securityAttr] = strings.TrimSpace(v[len(securityAttr):])
						} else if isExistsScope(securityAttr) {
							scopes[getScopeScheme(securityAttr)] = v[len(securityAttr):]
						}
						// next securityDefinitions
						if strings.Index(securityAttr, "@securitydefinitions.") == 0 {
							break
						}
					}
					if len(attrMap) != 2 {
						return errors.New("@securitydefinitions.oauth2.accessCode is @tokenUrl and @authorizationUrl required")
					}
					securityScheme := genswagger.OAuth2AccessToken(attrMap["@authorizationurl"], attrMap["@tokenurl"])
					for scope, description := range scopes {
						securityScheme.AddScope(scope, description)
					}
					securityMap[strings.TrimSpace(comments[i][len(attribute):])] = securityScheme
				}
			}
		}
	}
	if len(securityMap) > 0 {
		parser.swagger.SecurityDefinitions = securityMap
	}

	return nil
}

func getScopeScheme(scope string) string {
	scopeValue := scope[strings.Index(scope, "@scope."):]
	if scopeValue == "" {
		panic("@scope is empty")
	}
	return scope[len("@scope."):]
}

func isExistsScope(scope string) bool {
	s := strings.Fields(scope)
	for _, v := range s {
		if strings.Index(v, "@scope.") != -1 {
			if strings.Index(v, ",") != -1 {
				panic("@scope can't use comma(,) get=" + v)
			}
		}
	}
	return strings.Index(scope, "@scope.") != -1
}

// GetSchemes parses swagger schemes for given commentLine
func GetSchemes(commentLine string) []string {
	attribute := strings.ToLower(strings.Split(commentLine, " ")[0])
	return strings.Split(strings.TrimSpace(commentLine[len(attribute):]), " ")
}

func (parser *Parser) getNameFromPath(path string) string {
	if paths := strings.Split(path, "/"); len(paths) > 1 {
		return paths[len(paths)-1]
	}
	return path
}

// ParseRouterAPIInfo parses router api info for given astFile
func (parser *Parser) ParseRouterAPIInfo(fileName string, astFile *ast.File, info *parseCodenameInfo) error {
	for _, astDescription := range astFile.Decls {
		switch astDeclaration := astDescription.(type) {
		case *ast.FuncDecl:
			if astDeclaration.Name == nil || astDeclaration.Name.Name != "ServeHTTP" {
				continue
			}
			handlerName := parser.normalHandlerName(fileName, getStructName(astDeclaration), info)
			if _, ok := parser.HandlerName[handlerName]; !ok {
				continue
			}
			if !parser.checkNamespace(astDeclaration) {
				continue
			}
			operation := NewOperation()
			operation.parser = parser
			routeInfo := parser.HandlerName[handlerName]
			operation.Path = routeInfo.Pattern
			operation.HTTPMethod = routeInfo.Method
			operation.Operation.ID = strings.Replace(strings.ToLower(routeInfo.Name), " ", "-", -1)
			operation.Operation.Description = routeInfo.Description
			if routeInfo.AuthInfo.TokenType != "" {
				scopes := routeInfo.AuthInfo.RestrictScopes
				if scopes == nil {
					scopes = []string{}
				}
				var secure genswagger.SwaggerSecurityRequirementObject = map[string][]string{routeInfo.AuthInfo.TokenType: scopes}
				if operation.Operation.Security == nil {
					operation.Operation.Security = []genswagger.SwaggerSecurityRequirementObject{secure}
				} else {
					operation.Operation.Security = append(operation.Operation.Security, secure)
				}
				if parser.swagger.SecurityDefinitions == nil {
					parser.swagger.SecurityDefinitions = make(genswagger.SwaggerSecurityDefinitionsObject)
				}
				if _, ok := parser.swagger.SecurityDefinitions[routeInfo.AuthInfo.TokenType]; !ok {
					if se, ok := authDefine[routeInfo.AuthInfo.TokenType]; ok {
						parser.swagger.SecurityDefinitions[routeInfo.AuthInfo.TokenType] = se
					}
				}
			}
			var pathItem genswagger.SwaggerPathItemObject
			var oldCommentType string
			if astDeclaration.Doc != nil && astDeclaration.Doc.List != nil {
				for _, comment := range astDeclaration.Doc.List {
					var err error
					if oldCommentType, err = operation.ParseComment(comment.Text, astFile, oldCommentType); err != nil {
						return fmt.Errorf("ParseComment error in file %s :%+v", fileName, err)
					}
				}
			}
			var ok bool
			if pathItem, ok = parser.swagger.Paths[operation.Path]; !ok {
				pathItem = genswagger.SwaggerPathItemObject{}
			}
			if len(operation.Operation.Tags) < 1 && info != nil {
				operation.Operation.Tags = info.tags
			}
			if len(operation.Operation.Produces) < 1 {
				operation.Operation.Produces = []string{"application/json"}
			}
			switch strings.ToUpper(operation.HTTPMethod) {
			case http.MethodGet:
				pathItem.Get = &operation.Operation
			case http.MethodPost:
				pathItem.Post = &operation.Operation
			case http.MethodDelete:
				pathItem.Delete = &operation.Operation
			case http.MethodPut:
				pathItem.Put = &operation.Operation
			case http.MethodPatch:
				pathItem.Patch = &operation.Operation
			}
			parser.swagger.Paths[operation.Path] = pathItem
		}
	}

	return nil
}

func getStructName(funcDel *ast.FuncDecl) string {
	if recv := funcDel.Recv; recv != nil && len(recv.List) > 0 {
		for _, el := range recv.List {
			if names := el.Names; names != nil && len(names) > 0 {
				for _, name := range names {
					if obj := name.Obj; obj != nil {
						if decl := obj.Decl; decl != nil {
							if field, ok := decl.(*ast.Field); ok {
								if fieldType := field.Type; fieldType != nil {
									if fieldStarExpr, ok := fieldType.(*ast.StarExpr); ok {
										if fieldStarExprX := fieldStarExpr.X; fieldStarExprX != nil {
											if ident, ok := fieldStarExprX.(*ast.Ident); ok {
												return ident.Name
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return ""
}

func (parser *Parser) isInStructStack(refTypeName string) bool {
	for _, structName := range parser.structStack {
		if refTypeName == structName {
			return true
		}
	}
	return false
}

type structField struct {
	name         string
	schemaType   string
	arrayType    string
	formatType   string
	isRequired   bool
	crossPkg     string
	path         string
	exampleValue interface{}
	maximum      *float64
	minimum      *float64
	maxLength    *int64
	minLength    *int64
	description  string
	enums        []interface{}
	defaultValue interface{}
	extensions   map[string]interface{}
}

func floatPointerToV(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func intPointerToV(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func replaceLastTag(slice []genswagger.SwaggerTag, element genswagger.SwaggerTag) {
	slice = slice[:len(slice)-1]
	slice = append(slice, element)
}

func getFloatTag(structTag map[string]string, tagName string) *float64 {
	strValue, _ := structTag[tagName]
	if strValue == "" {
		strValue = structTag[strings.ToLower(tagName)]
		if strValue == "" {
			return nil
		}
	}

	value, err := strconv.ParseFloat(strValue, 64)
	if err != nil {
		panic(fmt.Errorf("can't parse numeric value of %q tag: %v", tagName, err))
	}

	return &value
}

func getIntTag(structTag map[string]string, tagName string) *int64 {
	strValue, _ := structTag[tagName]
	if strValue == "" {
		strValue = structTag[strings.ToLower(tagName)]
		if strValue == "" {
			return nil
		}
	}

	value, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		panic(fmt.Errorf("can't parse numeric value of %q tag: %v", tagName, err))
	}

	return &value
}

func toSnakeCase(in string) string {
	runes := []rune(in)
	length := len(runes)

	var out []rune
	for i := 0; i < length; i++ {
		if i > 0 && unicode.IsUpper(runes[i]) && ((i+1 < length && unicode.IsLower(runes[i+1])) || unicode.IsLower(runes[i-1])) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(runes[i]))
	}
	return string(out)
}

func toLowerCamelCase(in string) string {
	runes := []rune(in)

	var out []rune
	flag := false
	for i, curr := range runes {
		if (i == 0 && unicode.IsUpper(curr)) || (flag && unicode.IsUpper(curr)) {
			out = append(out, unicode.ToLower(curr))
			flag = true
		} else {
			out = append(out, curr)
			flag = false
		}
	}

	return string(out)
}

// defineTypeOfExample example value define the type (object and array unsupported)
func defineTypeOfExample(schemaType, arrayType, exampleValue string) interface{} {
	switch schemaType {
	case "string":
		return exampleValue
	case "number":
		v, err := strconv.ParseFloat(exampleValue, 64)
		if err != nil {
			fmt.Printf("example value %s can't convert to %s err: %s\n", exampleValue, schemaType, err)
			return nil
		}
		return v
	case "integer":
		v, err := strconv.Atoi(exampleValue)
		if err != nil {
			fmt.Printf("\nERROR - example value %s can't convert to %s err: %s\n", exampleValue, schemaType, err)
			return nil
		}
		return v
	case "boolean":
		v, err := strconv.ParseBool(exampleValue)
		if err != nil {
			fmt.Printf("\nERROR - example value %s can't convert to %s err: %s\n", exampleValue, schemaType, err)
			return nil
		}
		return v
	case "array":
		values := strings.Split(exampleValue, ",")
		result := make([]interface{}, 0)
		for _, value := range values {
			if ex := defineTypeOfExample(arrayType, "", value); ex != nil {
				result = append(result, ex)
			}
		}
		return result
	default:
		fmt.Printf("\nERROR - %s is unsupported type in example value\n", schemaType)
		return nil
	}
}

// GetAllGoFileInfo gets all Go source files information for given searchDir.
func (parser *Parser) getAllGoFileInfo(searchDir string) error {
	return filepath.Walk(searchDir, parser.visit)
}

func (parser *Parser) visit(path string, f os.FileInfo, err error) error {
	if err != nil {
		fmt.Println("ERROR - visit path:", path, "- err:", err)
		return nil
	}
	if f == nil {
		fmt.Println("WARN - path:", path, "not found")
		return nil
	}
	if err := Skip(f); err != nil {
		return err
	}

	if ext := filepath.Ext(path); ext == ".go" {
		fset := token.NewFileSet() // positions are relative to fset
		astFile, err := goparser.ParseFile(fset, path, nil, goparser.ParseComments)
		if err != nil {
			return fmt.Errorf("ParseFile error:%+v", err)
		}

		parser.files[path] = astFile
	}
	return nil
}

// Skip returns filepath.SkipDir error if match vendor and hidden folder
func Skip(f os.FileInfo) error {
	// exclude vendor folder
	if f.IsDir() && f.Name() == "vendor" {
		return filepath.SkipDir
	}

	// exclude all hidden folder
	if f.IsDir() && len(f.Name()) > 1 && f.Name()[0] == '.' {
		return filepath.SkipDir
	}
	return nil
}

// GetSwagger returns *spec.Swagger which is the root document object for the API specification.
func (parser *Parser) GetSwagger() *genswagger.SwaggerObject {
	return parser.swagger
}

func (p *Parser) normalHandlerName(fileName, handlerName string, info *parseCodenameInfo) string {
	if info != nil {
		if pkgStore, pkgOk := p.filePathPkgStore[info.name]; pkgOk {
			if filepath.Ext(fileName) == ".go" {
				fileNameSplit := strings.Split(fileName, "/")
				tmp := strings.Join(fileNameSplit[0:len(fileNameSplit)-1], "/")
				if handlerPkgName, hOk := pkgStore[tmp]; hOk {
					return fmt.Sprintf("%s.%s.%s", info.name, handlerPkgName, handlerName)
				}
			} else if handlerPkgName, hOk := pkgStore[fileName]; hOk {
				return fmt.Sprintf("%s.%s.%s", info.name, handlerPkgName, handlerName)
			}
		}
	}
	return handlerName
}
