package swag

import (
	"github.com/punky97/go-codebase/core/libs/docgen/genswagger"
	"fmt"
	"go/ast"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Operation describes a single API operation on a path.
// For more information: https://github.com/swaggo/swag#api-operation
type Operation struct {
	HTTPMethod string
	Path       string
	Operation  genswagger.SwaggerOperationObject

	parser *Parser
}

// Regular expression for comment with response
const responseCommentPattern = `([\d]+)[\s]+([\w\{\}]+)[\s]+([\w\-\.\/\(\)]+)[^"]*(.*)?`

var mimeTypeAliases = map[string]string{
	"json":                  "application/json",
	"xml":                   "text/xml",
	"plain":                 "text/plain",
	"html":                  "text/html",
	"mpfd":                  "multipart/form-data",
	"x-www-form-urlencoded": "application/x-www-form-urlencoded",
	"json-api":              "application/vnd.api+json",
	"json-stream":           "application/x-json-stream",
	"octet-stream":          "application/octet-stream",
	"png":                   "image/png",
	"jpeg":                  "image/jpeg",
	"gif":                   "image/gif",
}

var mimeTypePattern = regexp.MustCompile("^[^/]+/[^/]+$")

// NewOperation creates a new Operation with default properties.
// map[int]Response
func NewOperation() *Operation {
	return &Operation{
		HTTPMethod: "get",
		Operation:  genswagger.SwaggerOperationObject{},
	}
}

// ParseComment parses comment for given comment string and returns error if error occurs.
func (operation *Operation) ParseComment(comment string, astFile *ast.File, oldTypeComment string) (typeComment string, err error) {
	commentLine := strings.TrimSpace(strings.TrimLeft(comment, "//"))

	if len(commentLine) == 0 || commentLine[0] != '@' {
		switch oldTypeComment {
		case "description":
			if operation.Operation.Description != "" {
				operation.Operation.Description += "\n\n" + strings.Replace(commentLine, "\\n", "\n", -1)
				return "description", nil
			}
		}
		return "", nil
	}

	attribute := strings.Fields(commentLine)[0]
	lineRemainder := strings.TrimSpace(commentLine[len(attribute):])
	switch strings.ToLower(attribute) {
	case "@description":
		if operation.Operation.Description == "" {
			operation.Operation.Description = strings.Replace(lineRemainder, "\\n", "\n", -1)
		} else {
			operation.Operation.Description += "\n\n" + strings.Replace(lineRemainder, "\\n", "\n", -1)
		}
		typeComment = "description"
	case "@summary":
		operation.Operation.Summary = lineRemainder
		typeComment = "summary"
	case "@id":
		operation.Operation.ID = lineRemainder
	case "@tags":
		operation.ParseTagsComment(lineRemainder)
	case "@accept":
		if err := operation.ParseAcceptComment(lineRemainder); err != nil {
			return "", err
		}
	case "@produce":
		if err := operation.ParseProduceComment(lineRemainder); err != nil {
			return "", err
		}
	case "@param":
		if err := operation.ParseParamComment(lineRemainder, astFile); err != nil {
			return "", err
		}
	case "@success", "@failure":
		if err := operation.ParseResponseComment(lineRemainder, astFile); err != nil {
			if err := operation.ParseEmptyResponseComment(lineRemainder); err != nil {
				if err := operation.ParseEmptyResponseOnly(lineRemainder); err != nil {
					return "", err
				}
			}
		}
	case "@header":
		if err := operation.ParseResponseHeaderComment(lineRemainder, astFile); err != nil {
			return "", err
		}
	case "@router":
		if err := operation.ParseRouterComment(strings.TrimSpace(commentLine[len(attribute):])); err != nil {
			return "", err
		}
	case "@security":
		if err := operation.ParseSecurityComment(strings.TrimSpace(commentLine[len(attribute):])); err != nil {
			return "", err
		}
	}
	return typeComment, nil
}

// ParseParamComment parses params return []string of param properties
// @Param	queryText		form	      string	  true		        "The email for login"
// 			[param name]    [paramType] [data type]  [is mandatory?]   [Comment]
// @Param   some_id     path    int     true        "Some ID"
func (operation *Operation) ParseParamComment(commentLine string, astFile *ast.File) error {
	re := regexp.MustCompile(`(\S+)[\s]+([\w]+)[\s]+([\S.]+)[\s]+([\w]+)[\s]+"([^"]+)"`)
	matches := re.FindStringSubmatch(commentLine)
	if len(matches) != 6 {
		return fmt.Errorf("missing required param comment parameters \"%s\"", commentLine)
	}
	name := matches[1]
	paramType := matches[2]

	schemaType := matches[3]

	requiredText := strings.ToLower(matches[4])
	required := requiredText == "true" || requiredText == "required"
	description := matches[5]

	var param genswagger.SwaggerParameterObject

	//five possible parameter types.
	switch paramType {
	case "query", "path", "header":
		param = createParameter(paramType, description, name, TransToValidSchemeType(schemaType), required)
	case "body":
		param = createParameter(paramType, description, name, "object", required)
		if schemaNameToPath, ok := operation.parser.schemaSortName2FullPath[operation.Path]; ok {
			if namePath, ok := schemaNameToPath[schemaType]; ok {
				param.Schema = &genswagger.Type{
					Description: param.Description,
				}
				param.Schema.Ref = "#/definitions/" + operation.parser.trueNameDefinition[namePath]
			}
		}
	case "formData":
		param = createParameter(paramType, description, name, TransToValidSchemeType(schemaType), required)
	default:
		return fmt.Errorf("%s is not supported paramType", paramType)
	}

	if err := operation.parseAndExtractionParamAttribute(commentLine, schemaType, &param); err != nil {
		return err
	}
	operation.Operation.Parameters = append(operation.Operation.Parameters, param)
	return nil
}

var regexAttributes = map[string]*regexp.Regexp{
	// for Enums(A, B)
	"enums": regexp.MustCompile(`(?i)enums\(.*\)`),
	// for Minimum(0)
	"maxinum": regexp.MustCompile(`(?i)maxinum\(.*\)`),
	// for Maximum(0)
	"mininum": regexp.MustCompile(`(?i)mininum\(.*\)`),
	// for Maximum(0)
	"default": regexp.MustCompile(`(?i)default\(.*\)`),
	// for minlength(0)
	"minlength": regexp.MustCompile(`(?i)minlength\(.*\)`),
	// for maxlength(0)
	"maxlength": regexp.MustCompile(`(?i)maxlength\(.*\)`),
	// for format(email)
	"format": regexp.MustCompile(`(?i)format\(.*\)`),
}

func (operation *Operation) parseAndExtractionParamAttribute(commentLine, schemaType string, param *genswagger.SwaggerParameterObject) error {
	schemaType = TransToValidSchemeType(schemaType)
	for attrKey, re := range regexAttributes {
		switch attrKey {
		case "enums":
			attr := re.FindString(commentLine)
			l := strings.Index(attr, "(")
			r := strings.Index(attr, ")")
			if !(l == -1 && r == -1) {
				enums := strings.Split(attr[l+1:r], ",")
				for _, e := range enums {
					e = strings.TrimSpace(e)
					param.Enum = append(param.Enum, defineType(schemaType, e))
				}
			}
		case "maxinum":
			attr := re.FindString(commentLine)
			l := strings.Index(attr, "(")
			r := strings.Index(attr, ")")
			if !(l == -1 && r == -1) {
				if schemaType != "integer" && schemaType != "number" {
					return fmt.Errorf("maxinum is attribute to set to a number. comment=%s got=%s", commentLine, schemaType)
				}
				attr = strings.TrimSpace(attr[l+1 : r])
				n, err := strconv.ParseFloat(attr, 64)
				if err != nil {
					return fmt.Errorf("maximum is allow only a number. comment=%s got=%s", commentLine, attr)
				}
				param.Maximum = &n
			}
		case "mininum":
			attr := re.FindString(commentLine)
			l := strings.Index(attr, "(")
			r := strings.Index(attr, ")")
			if !(l == -1 && r == -1) {
				if schemaType != "integer" && schemaType != "number" {
					return fmt.Errorf("mininum is attribute to set to a number. comment=%s got=%s", commentLine, schemaType)
				}
				attr = strings.TrimSpace(attr[l+1 : r])
				n, err := strconv.ParseFloat(attr, 64)
				if err != nil {
					return fmt.Errorf("mininum is allow only a number got=%s", attr)
				}
				param.Minimum = &n
			}
		case "default":
			attr := re.FindString(commentLine)
			l := strings.Index(attr, "(")
			r := strings.Index(attr, ")")
			if !(l == -1 && r == -1) {
				attr = strings.TrimSpace(attr[l+1 : r])
				param.Default = defineType(schemaType, attr)
			}
		case "maxlength":
			attr := re.FindString(commentLine)
			l := strings.Index(attr, "(")
			r := strings.Index(attr, ")")
			if !(l == -1 && r == -1) {
				if schemaType != "string" {
					return fmt.Errorf("maxlength is attribute to set to a number. comment=%s got=%s", commentLine, schemaType)
				}
				attr = strings.TrimSpace(attr[l+1 : r])
				n, err := strconv.ParseInt(attr, 10, 64)
				if err != nil {
					return fmt.Errorf("maxlength is allow only a number got=%s", attr)
				}
				param.MaxLength = &n
			}
		case "minlength":
			attr := re.FindString(commentLine)
			l := strings.Index(attr, "(")
			r := strings.Index(attr, ")")
			if !(l == -1 && r == -1) {
				if schemaType != "string" {
					return fmt.Errorf("maxlength is attribute to set to a number. comment=%s got=%s", commentLine, schemaType)
				}
				attr = strings.TrimSpace(attr[l+1 : r])
				n, err := strconv.ParseInt(attr, 10, 64)
				if err != nil {
					return fmt.Errorf("minlength is allow only a number got=%s", attr)
				}
				param.MinLength = &n
			}
		case "format":
			attr := re.FindString(commentLine)
			l := strings.Index(attr, "(")
			r := strings.Index(attr, ")")
			if !(l == -1 && r == -1) {
				param.Format = strings.TrimSpace(attr[l+1 : r])
			}
		}
	}
	return nil
}

// defineType enum value define the type (object and array unsupported)
func defineType(schemaType string, value string) interface{} {
	schemaType = TransToValidSchemeType(schemaType)
	switch schemaType {
	case "string":
		return value
	case "number":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			panic(fmt.Errorf("enum value %s can't convert to %s err: %s", value, schemaType, err))
		}
		return v
	case "integer":
		v, err := strconv.Atoi(value)
		if err != nil {
			panic(fmt.Errorf("enum value %s can't convert to %s err: %s", value, schemaType, err))
		}
		return v
	case "boolean":
		v, err := strconv.ParseBool(value)
		if err != nil {
			panic(fmt.Errorf("enum value %s can't convert to %s err: %s", value, schemaType, err))
		}
		return v
	default:
		panic(fmt.Errorf("%s is unsupported type in enum value", schemaType))
	}
}

// ParseTagsComment parses comment for given `tag` comment string.
func (operation *Operation) ParseTagsComment(commentLine string) {
	tags := strings.Split(commentLine, ",")
	for _, tag := range tags {
		operation.Operation.Tags = append(operation.Operation.Tags, strings.TrimSpace(tag))
	}
}

// ParseAcceptComment parses comment for given `accept` comment string.
func (operation *Operation) ParseAcceptComment(commentLine string) error {
	return parseMimeTypeList(commentLine, &operation.Operation.Consumes, "%v accept type can't be accepted")
}

// ParseProduceComment parses comment for given `produce` comment string.
func (operation *Operation) ParseProduceComment(commentLine string) error {
	return parseMimeTypeList(commentLine, &operation.Operation.Produces, "%v produce type can't be accepted")
}

// parseMimeTypeList parses a list of MIME Types for a comment like
// `produce` (`Content-Type:` response header) or
// `accept` (`Accept:` request header)
func parseMimeTypeList(mimeTypeList string, typeList *[]string, format string) error {
	mimeTypes := strings.Split(mimeTypeList, ",")
	for _, typeName := range mimeTypes {
		if mimeTypePattern.MatchString(typeName) {
			*typeList = append(*typeList, typeName)
		} else if aliasMimeType, ok := mimeTypeAliases[typeName]; ok {
			*typeList = append(*typeList, aliasMimeType)
		} else {
			return fmt.Errorf(format, typeName)
		}
	}
	return nil
}

// ParseRouterComment parses comment for gived `router` comment string.
func (operation *Operation) ParseRouterComment(commentLine string) error {
	re := regexp.MustCompile(`([\w\.\/\-{}\+]+)[^\[]+\[([^\]]+)`)
	var matches []string

	if matches = re.FindStringSubmatch(commentLine); len(matches) != 3 {
		return fmt.Errorf("can not parse router comment \"%s\"", commentLine)
	}
	path := matches[1]
	httpMethod := matches[2]

	operation.Path = path
	operation.HTTPMethod = strings.ToUpper(httpMethod)

	return nil
}

// ParseSecurityComment parses comment for gived `security` comment string.
func (operation *Operation) ParseSecurityComment(commentLine string) error {
	securitySource := commentLine[strings.Index(commentLine, "@Security")+1:]
	l := strings.Index(securitySource, "[")
	r := strings.Index(securitySource, "]")
	// exists scope
	if !(l == -1 && r == -1) {
		scopes := securitySource[l+1 : r]
		s := []string{}
		for _, scope := range strings.Split(scopes, ",") {
			scope = strings.TrimSpace(scope)
			s = append(s, scope)
		}
		securityKey := securitySource[0:l]
		securityMap := map[string][]string{}
		securityMap[securityKey] = append(securityMap[securityKey], s...)
		operation.Operation.Security = append(operation.Operation.Security, securityMap)
	} else {
		securityKey := strings.TrimSpace(securitySource)
		securityMap := map[string][]string{}
		securityMap[securityKey] = []string{}
		operation.Operation.Security = append(operation.Operation.Security, securityMap)
	}
	return nil
}

// ParseResponseComment parses comment for gived `response` comment string.
func (operation *Operation) ParseResponseComment(commentLine string, astFile *ast.File) error {
	re := regexp.MustCompile(responseCommentPattern)
	var matches []string

	if matches = re.FindStringSubmatch(commentLine); len(matches) != 5 {
		return fmt.Errorf("can not parse response comment \"%s\"", commentLine)
	}

	response := genswagger.SwaggerResponseObject{}

	code, _ := strconv.Atoi(matches[1])

	responseDescription := strings.Trim(matches[4], "\"")
	if responseDescription == "" {
		responseDescription = http.StatusText(code)
	}
	response.Description = responseDescription

	schemaType := strings.Trim(matches[2], "{}")
	refType := matches[3]

	response.Schema = &genswagger.Type{
		Type: schemaType,
	}
	if schemaNameToPath, ok := operation.parser.schemaSortName2FullPath[operation.Path]; ok {
		if namePath, ok := schemaNameToPath[refType]; ok {
			refType = operation.parser.trueNameDefinition[namePath]
		}
	}

	if schemaType == "object" {
		response.Schema.Type = ""
		response.Schema.Ref = "#/definitions/" + refType
	}

	if schemaType == "array" {
		refType = TransToValidSchemeType(refType)
		if IsPrimitiveType(refType) {
			response.Schema.Items = &genswagger.Type{
				Type: refType,
			}
		} else {
			response.Schema.Items = &genswagger.Type{
				Ref: "#/definitions/" + refType,
			}
		}
	}

	if operation.Operation.Responses == nil {
		operation.Operation.Responses = make(map[int]genswagger.SwaggerResponseObject)
	}

	operation.Operation.Responses[code] = response
	return nil
}

// ParseResponseHeaderComment parses comment for gived `response header` comment string.
func (operation *Operation) ParseResponseHeaderComment(commentLine string, astFile *ast.File) error {
	re := regexp.MustCompile(responseCommentPattern)
	var matches []string

	if matches = re.FindStringSubmatch(commentLine); len(matches) != 5 {
		return fmt.Errorf("can not parse response comment \"%s\"", commentLine)
	}

	response := genswagger.SwaggerResponseObject{}

	code, _ := strconv.Atoi(matches[1])

	responseDescription := strings.Trim(matches[4], "\"")
	if responseDescription == "" {
		responseDescription = http.StatusText(code)
	}
	response.Description = responseDescription

	schemaType := strings.Trim(matches[2], "{}")
	refType := matches[3]

	if operation.Operation.Responses == nil {
		operation.Operation.Responses = make(map[int]genswagger.SwaggerResponseObject)
	}

	response, responseExist := operation.Operation.Responses[code]
	if responseExist {
		header := genswagger.Header{}
		header.Description = responseDescription
		header.Type.Type = schemaType

		if response.Headers == nil {
			response.Headers = make(map[string]genswagger.Header)
		}
		response.Headers[refType] = header

		operation.Operation.Responses[code] = response
	}

	return nil
}

// ParseEmptyResponseComment parse only comment out status code and description,eg: @Success 200 "it's ok"
func (operation *Operation) ParseEmptyResponseComment(commentLine string) error {
	re := regexp.MustCompile(`([\d]+)[\s]+"(.*)"`)
	var matches []string

	if matches = re.FindStringSubmatch(commentLine); len(matches) != 3 {
		return fmt.Errorf("can not parse response comment \"%s\"", commentLine)
	}

	response := genswagger.SwaggerResponseObject{}

	code, _ := strconv.Atoi(matches[1])

	response.Description = strings.Trim(matches[2], "")

	if operation.Operation.Responses == nil {
		operation.Operation.Responses = make(map[int]genswagger.SwaggerResponseObject)
	}

	operation.Operation.Responses[code] = response

	return nil
}

//ParseEmptyResponseOnly parse only comment out status code ,eg: @Success 200
func (operation *Operation) ParseEmptyResponseOnly(commentLine string) error {
	response := genswagger.SwaggerResponseObject{}

	code, err := strconv.Atoi(commentLine)
	if err != nil {
		return fmt.Errorf("can not parse response comment \"%s\"", commentLine)
	}
	if operation.Operation.Responses == nil {
		operation.Operation.Responses = make(map[int]genswagger.SwaggerResponseObject)
	}

	operation.Operation.Responses[code] = response

	return nil
}

// createParameter returns swagger spec.Parameter for gived  paramType, description, paramName, schemaType, required
func createParameter(paramType, description, paramName, schemaType string, required bool) genswagger.SwaggerParameterObject {
	// //five possible parameter types. 	query, path, body, header, form
	paramProps := genswagger.SwaggerParameterObject{
		Name:        paramName,
		Description: description,
		Required:    required,
		In:          paramType,
		Type:        schemaType,
	}
	return paramProps
}

func (operation *Operation) GetDefinedFromParamsComment(commentLine string, astFile *ast.File) (string, error) {
	re := regexp.MustCompile(`(\S+)[\s]+([\w]+)[\s]+([\S.]+)[\s]+([\w]+)[\s]+"([^"]+)"`)
	matches := re.FindStringSubmatch(commentLine)
	if len(matches) != 6 {
		return "", fmt.Errorf("missing required param comment parameters \"%s\"", commentLine)
	}
	schemaType := matches[2]
	switch schemaType {
	case "body":
		return matches[3], nil
	}
	return "", nil
}

// GetDefineFromResponseComment parses comment for gived `response` comment string.
func (operation *Operation) GetDefineFromResponseComment(commentLine string, astFile *ast.File) (string, error) {
	re := regexp.MustCompile(responseCommentPattern)
	var matches []string

	if matches = re.FindStringSubmatch(commentLine); len(matches) != 5 {
		return "", fmt.Errorf("can not parse response comment \"%s\"", commentLine)
	}
	schemaType := strings.Trim(matches[2], "{}")
	switch schemaType {
	case "object":
		return matches[3], nil
	}
	return "", nil
}
