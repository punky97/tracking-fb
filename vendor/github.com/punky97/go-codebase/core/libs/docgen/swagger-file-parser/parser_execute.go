package swag

import (
	"fmt"
	"github.com/punky97/go-codebase/core/libs/docgen/genswagger"
	"github.com/punky97/go-codebase/core/utils"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var pathIgnores = []string{"context", "encoding", "net", "regexp", "fmt", "google.golang.org", "github.com", "time", "strings", "flag"}

// ParseAPI parses general api info for give searchDir (path of cmd) and mainAPIFile (main api path)
func (parser *Parser) ParseAPI(searchDir string, mainAPIFile string) (*genswagger.SwaggerObject, error) {
	searchDir = os.Getenv("GOPATH") + "/src/" + searchDir
	parser.pathParsed[searchDir] = true
	if err := parser.getAllGoFileInfo(searchDir); err != nil {
		return nil, err
	}
	for fileName, astFile := range parser.files {
		parser.GetDefNeedParseFromRouterAPI(fileName, astFile, nil)
	}
	parser.ParseGeneralAPIInfo(path.Join(searchDir, mainAPIFile))
	parser.mergeDefinitionsToSwagger()

	for fileName, astFile := range parser.files {
		if err := parser.ParseRouterAPIInfo(fileName, astFile, nil); err != nil {
			return nil, err
		}
	}

	return parser.swagger, nil
}

// GetDefNeedParseFromRouterAPI check in handler file and find struct must define and path of this
func (parser *Parser) GetDefNeedParseFromRouterAPI(fileName string, astFile *ast.File, info *parseCodenameInfo) error {
	for _, astDescription := range astFile.Decls {
		switch astDeclaration := astDescription.(type) {
		case *ast.FuncDecl:
			if astDeclaration.Name == nil || astDeclaration.Name.Name != "ServeHTTP" {
				continue
			}
			handlerName := parser.normalHandlerName(fileName, getStructName(astDeclaration), info)
			var routeInfo genswagger.SwaggerRouteInfo
			var ok bool
			if routeInfo, ok = parser.HandlerName[handlerName]; !ok {
				continue
			}
			if !parser.checkNamespace(astDeclaration) {
				continue
			}
			if astDeclaration.Doc != nil && astDeclaration.Doc.List != nil {
				operation := NewOperation()
				operation.parser = parser
				var name string
				var err error
				for _, comment := range astDeclaration.Doc.List {
					commentLine := strings.TrimSpace(strings.TrimLeft(comment.Text, "//"))
					if len(commentLine) == 0 {
						continue
					}

					attribute := strings.Fields(commentLine)[0]
					lineRemainder := strings.TrimSpace(commentLine[len(attribute):])
					switch strings.ToLower(attribute) {
					case "@param":
						name, err = operation.GetDefinedFromParamsComment(lineRemainder, astFile)
					case "@success", "@failure":
						name, err = operation.GetDefineFromResponseComment(lineRemainder, astFile)
					}

					if err == nil && name != "" {
						schemaPath, schemaName := parser.getPathOfSchemaShouldParse(fileName, name, astFile)
						if parser.schemaSortName2FullPath[routeInfo.Pattern] == nil {
							parser.schemaSortName2FullPath[routeInfo.Pattern] = map[string]string{
								name: parser.pathAndName(schemaPath, schemaName),
							}
						} else {
							parser.schemaSortName2FullPath[routeInfo.Pattern][name] = parser.pathAndName(schemaPath, schemaName)
						}

						parser.ParseSchema(schemaPath, schemaName)
					}
				}
			}
		}
	}
	return nil
}

func (parser *Parser) checkNamespace(astDeclaration *ast.FuncDecl) bool {
	if parser.Namespace == "" {
		return true
	}
	if astDeclaration.Doc != nil && astDeclaration.Doc.List != nil {
		for _, comment := range astDeclaration.Doc.List {
			commentLine := strings.TrimSpace(strings.TrimLeft(comment.Text, "//"))
			if len(commentLine) == 0 {
				continue
			}

			attribute := strings.Fields(commentLine)[0]
			lineRemainder := strings.TrimSpace(commentLine[len(attribute):])
			if strings.ToLower(attribute) == "@namespace" {
				namespaces := GetNamespaceFromNamespaceComment(lineRemainder)
				if utils.IsStringSliceContains(namespaces, parser.Namespace) {
					return true
				} else {
					return false
				}
			}
		}
	}
	return false
}

func (parser *Parser) pathAndName(path, name string) string {
	return strings.Replace(strings.Trim(path, `" "`), "/", "/", -1) + "." + name
}

var rBeeDefinedStruct = regexp.MustCompile(`([a-zA-Z1-9]+)(\([a-zA-Z1-9_/]+\))$`)

// getPathOfSchemaShouldParse - get path of schema (exp: bkgo/exmsg/cbox/models.CboxSetting) when it define in route
func (parser *Parser) getPathOfSchemaShouldParse(filePath string, schemaName string, astFile *ast.File) (
	string, string) {
	if res := rBeeDefinedStruct.FindStringSubmatch(schemaName); len(res) >= 1 {
		return strings.Trim(res[2], `)(`), res[1]
	}
	schemaNameAndPath := strings.Split(schemaName, ".")
	if len(schemaNameAndPath) > 2 {
		fmt.Println("Error: wrong define ", schemaName, " in ", filePath)
	}
	if len(schemaNameAndPath) < 2 {
		filePath = parser.normalPath(filePath)
		if filepath.Ext(filePath) == ".go" {
			filePathSp := strings.Split(filePath, "/")
			filePathSp = filePathSp[:len(filePathSp)-1]
			filePath = strings.Join(filePathSp, "/")
		}

		return filePath, schemaName
	}
	for _, importPath := range astFile.Imports {
		if importPath.Path != nil {
			path := strings.Trim(importPath.Path.Value, `" `)
			if pathImportIgnore(path) {
				continue
			}
			if importPath.Name != nil {
				if importPath.Name.Name == schemaNameAndPath[0] {
					return path, schemaNameAndPath[1]
				}
			} else if getEndOfPath(path) == schemaNameAndPath[0] {
				return path, schemaNameAndPath[1]
			} else if parser.checkSamePkgAndImport(path, schemaNameAndPath[0]) {
				return path, schemaNameAndPath[1]
			}
		}
	}
	fmt.Println("Error: not found ", schemaName, " from import in ", filePath)
	panic("error field: " + schemaName)
}

func pathImportIgnore(path string) bool {
	sp := strings.Split(path, "/")
	if utils.IsStringSliceContains(pathIgnores, sp[0]) {
		return true
	}
	return false
}

// exmsg/core/models -> models
func getEndOfPath(path string) string {
	pathSp := strings.Split(path, "/")
	return pathSp[len(pathSp)-1]
}

// in some case this package name is defined by `package core_models` is not same the path so it should parse all file and check
func (parser *Parser) checkSamePkgAndImport(path string, pkgName string) bool {
	path = parser.basePath + strings.Replace(strings.Trim(path, `" "`), "/", "/", -1)
	files, _ := parser.getFilesFromPath(path)
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".go" {
			fileGoPath := path + "/" + file.Name()
			astFile := parser.getAstFileFromFilePath(fileGoPath)
			if astFile.Name != nil && astFile.Name.Name == pkgName {
				return true
			}
		}
	}
	return false
}

// get all file info file pkg path
func (parser *Parser) getFilesFromPath(path string) ([]os.FileInfo, error) {
	var err error
	var ok bool
	var files []os.FileInfo
	if files, ok = parser.pkgFileInfo[path]; !ok {
		files, err = ioutil.ReadDir(path)
		if err != nil {
			fmt.Println("Warn when open", path, "err:", err)
			return nil, err
		}
		parser.pkgFileInfo[path] = files
	}
	return files, err
}

func (parser *Parser) getAstFileFromFilePath(fileGoPath string) *ast.File {
	if astFile, ok := parser.files[fileGoPath]; ok {
		return astFile
	} else {
		fset := token.NewFileSet()
		astFile, err := goparser.ParseFile(fset, fileGoPath, nil, goparser.ParseComments)
		if err != nil {
			panic("Cannot open " + fileGoPath)
		}
		parser.files[fileGoPath] = astFile
		return astFile
	}
}

// normalPath - convert full path to relative path
func (parser *Parser) normalPath(path string) string {
	path = strings.Replace(strings.Trim(path, `"`), parser.basePath, "", 1)
	return path
}

func (parser *Parser) ParseSchema(path, schemaName string) *genswagger.Type {
	if parser.normalPath(path) == "time" && strings.ToLower(strings.Trim(schemaName, `" `)) == "time" {
		return &genswagger.Type{
			Type:   "string",
			Format: "time",
		}
	}
	path = parser.normalPath(path)
	err := parser.parseAllSchemaInPath(path)
	if err != nil {
		defpath := parser.pathAndName(path, schemaName)
		parser.schemaParsedDefinitions[defpath] = true
		return &genswagger.Type{
			Type: "object",
		}
	}
	tmp := parser.ParseDefinition(path, schemaName)
	if tmp == nil {
		if _, ok := parser.CustomPrimitiveTypes[schemaName]; ok {
			return nil
		}
	}
	if tmp != nil && tmp.Example != nil {
		fmt.Println("example:", tmp.Example)
	}
	if tmp == nil {
		fmt.Println("[ERROR] cannot find:", schemaName, "in path", path)
		panic("not found schema name")
	}
	return tmp
}

// ParseDefinition parses given type spec that corresponds to the type under
// given name and package, and populates swagger schema definitions registry
// with a schema for the given type
func (parser *Parser) ParseDefinition(path, typeName string) *genswagger.Type {
	defpath := parser.pathAndName(path, typeName)
	if typeDef, ok := parser.Definitions[defpath]; ok {
		return typeDef
	}
	parser.schemaParsedDefinitions[defpath] = true
	astFile := parser.schemaInFile[path][typeName]
	typeSpec := parser.fullPathDefinitions[path][typeName]
	if typeSpec == nil {
		return nil
	}
	typ := parser.parseTypeExpr(path, typeName, typeSpec.Type, astFile)
	parser.Definitions[defpath] = typ
	return typ
}

func (parser *Parser) parseAllSchemaInPath(path string) error {
	path = parser.basePath + strings.Replace(strings.Trim(path, `" "`), "/", "/", -1)
	files, err := parser.getFilesFromPath(path)
	if err != nil {
		return err
	}
	if _, ok := parser.pathParsed[path]; ok {
		return nil
	}
	parser.pathParsed[path] = true

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".go" {
			fileGoPath := path + "/" + file.Name()
			astFile := parser.getAstFileFromFilePath(fileGoPath)
			parser.ParseTypeFullPath(astFile, path)
		}
	}
	return nil
}

// ParseType parses type info for given astFile.
func (parser *Parser) ParseTypeFullPath(astFile *ast.File, path string) {
	for _, astDeclaration := range astFile.Decls {
		if generalDeclaration, ok := astDeclaration.(*ast.GenDecl); ok && generalDeclaration.Tok == token.TYPE {
			for _, astSpec := range generalDeclaration.Specs {
				if typeSpec, ok := astSpec.(*ast.TypeSpec); ok {
					typeName := fmt.Sprintf("%v", typeSpec.Type)
					// check if its a custom primitive type
					if IsGolangPrimitiveType(typeName) {
						parser.CustomPrimitiveTypes[typeSpec.Name.String()] = TransToValidSchemeType(typeName)
					} else {
						normalpath := parser.normalPath(path)
						if parser.fullPathDefinitions[normalpath] == nil {
							parser.fullPathDefinitions[normalpath] = make(map[string]*ast.TypeSpec)
							parser.schemaInFile[normalpath] = make(map[string]*ast.File)
						}
						parser.fullPathDefinitions[normalpath][typeSpec.Name.String()] = typeSpec
						parser.schemaInFile[normalpath][typeSpec.Name.String()] = astFile
					}
				}
			}
		}
	}
}

// ParseDefinitionByPath parses given type spec that corresponds to the type under
// given name and path, and populates swagger schema definitions registry
// with a schema for the given type
func (parser *Parser) ParseDefinitionByPath(path, typeName string) {
	if _, ok := parser.Definitions[path+"."+typeName]; ok {
		return
	}
	if pathDef, ok := parser.fullPathDefinitions[path]; ok {
		if _, ok := pathDef[typeName]; ok {

		}
	} else {
		parser.Definitions[path+"."+typeName] = &genswagger.Type{
			Type: "object",
		}
	}

}

// parseTypeExpr parses given type expression that corresponds to the type under
// given name and package, and returns swagger schema for it.
func (parser *Parser) parseTypeExpr(path, typeName string, typeExpr ast.Expr, astFile *ast.File) *genswagger.Type {
	switch expr := typeExpr.(type) {
	// type Foo struct {...}
	case *ast.StructType:
		refTypeName := parser.pathAndName(path, typeName)
		if schema, isParsed := parser.swagger.Definitions[refTypeName]; isParsed {
			return schema
		}

		extraRequired := make([]string, 0)
		properties := make(map[string]*genswagger.Type)
		for _, field := range expr.Fields.List {
			var fieldProps map[string]*genswagger.Type
			var requiredFromAnon []string
			if field.Names == nil {
				fieldProps, requiredFromAnon = parser.parseAnonymousField(path, field, astFile)
				extraRequired = append(extraRequired, requiredFromAnon...)
			} else {
				fieldProps = parser.parseStruct(path, field, astFile)
			}

			for k, v := range fieldProps {
				//if v.Example != nil {
				//	fmt.Println(path, typeName, k, v.Example)
				//}
				properties[k] = v
			}
		}

		// collect requireds from our properties and anonymous fields
		required := parser.collectRequiredFields(path, properties, extraRequired)

		// unset required from properties because we've collected them
		for k, prop := range properties {
			tname := prop.Type
			if tname != "object" {
				prop.Required = make([]string, 0)
			}
			properties[k] = prop
		}

		return &genswagger.Type{
			Type:       "object",
			Properties: properties,
			Required:   required,
		}

		// type Foo Baz
	case *ast.Ident:
		refTypeName := parser.pathAndName(path, expr.Name)
		if _, ok := parser.schemaParsedDefinitions[refTypeName]; !ok {
			parser.ParseSchema(path, expr.Name)
		}
		return parser.Definitions[refTypeName]

		// type Foo *Baz
	case *ast.StarExpr:
		return parser.parseTypeExpr(path, typeName, expr.X, astFile)

		// type Foo []Baz
	case *ast.ArrayType:
		itemSchema := parser.parseTypeExpr(path, "", expr.Elt, astFile)
		return &genswagger.Type{
			Type:  "array",
			Items: itemSchema,
		}

		// type Foo pkg.Bar
	case *ast.SelectorExpr:
		if xIdent, ok := expr.X.(*ast.Ident); ok {
			pkgName := xIdent.Name
			typeName = expr.Sel.Name
			schemaPath, typeName := parser.getPathOfSchemaShouldParse(path, pkgName+"."+typeName, astFile)
			refTypeName := parser.pathAndName(schemaPath, typeName)
			if _, ok := parser.schemaParsedDefinitions[refTypeName]; !ok {
				return parser.ParseSchema(schemaPath, typeName)
			}
			return parser.swagger.Definitions[parser.pathAndName(schemaPath, typeName)]
		}

		// type Foo map[string]Bar
	case *ast.MapType:
		itemSchema := parser.parseTypeExpr(path, "", expr.Value, astFile)
		return &genswagger.Type{
			Type:              "object",
			PatternProperties: itemSchema,
		}
		// ...
	default:
		fmt.Printf("Type definition of type '%T' is not supported yet. Using 'object' instead.\n", typeExpr)
	}

	return &genswagger.Type{
		Type: "object",
	}
}

// `type struct A { Id string }` in `type struct B {A}`
func (parser *Parser) parseAnonymousField(path string, field *ast.Field, astFile *ast.File) (map[string]*genswagger.Type, []string) {
	properties := make(map[string]*genswagger.Type)

	fullTypeName := ""
	switch ftype := field.Type.(type) {
	case *ast.Ident:
		fullTypeName = ftype.Name
	case *ast.StarExpr:
		if ftypeX, ok := ftype.X.(*ast.Ident); ok {
			fullTypeName = ftypeX.Name
		} else if ftypeX, ok := ftype.X.(*ast.SelectorExpr); ok {
			if packageX, ok := ftypeX.X.(*ast.Ident); ok {
				fullTypeName = fmt.Sprintf("%s.%s", packageX.Name, ftypeX.Sel.Name)
			}
		} else {
			fmt.Printf("Composite field type of '%T' is unhandle by parser. Skipping", ftype)
			return properties, []string{}
		}
	case *ast.SelectorExpr:
		if ftype.X != nil && ftype.Sel != nil {
			if ftypeX, ok := ftype.X.(*ast.Ident); ok {
				fullTypeName = fmt.Sprintf("%s.%s", ftypeX.Name, ftype.Sel.Name)
			} else {
				return properties, []string{}
			}
		} else {
			fmt.Println("*ast.SelectorExpr is unsupported, field name: ", ftype.Sel.Name)
			return properties, []string{}
		}
	default:
		fmt.Printf("Field type of '%T' is unsupported. Skipping", ftype)
		return properties, []string{}
	}

	typeName := fullTypeName
	path, typeName = parser.getPathOfSchemaShouldParse(path, typeName, astFile)
	schema := parser.ParseSchema(path, typeName)

	schemaType := "unknown"
	if len(schema.Type) > 0 {
		schemaType = schema.Type
	}

	switch schemaType {
	case "object":
		for k, v := range schema.Properties {
			properties[k] = v
		}
	case "array":
		properties[typeName] = schema
	default:
		fmt.Printf("Can't extract properties from a schema of type '%s'", schemaType)
	}

	return properties, schema.Required
}

func (parser *Parser) parseField(field *ast.Field, astFile *ast.File, filePath string) *structField {
	prop := getPropertyName(astFile, filePath, field, parser)
	if len(prop.ArrayType) == 0 {
		CheckSchemaType(prop.SchemaType)
	} else {
		CheckSchemaType("array")
	}
	structField := &structField{
		name:       field.Names[0].Name,
		schemaType: prop.SchemaType,
		arrayType:  prop.ArrayType,
		path:       prop.CrossPkg,
	}

	switch parser.PropNamingStrategy {
	case SnakeCase:
		structField.name = toSnakeCase(structField.name)
	case PascalCase:
		//use struct field name
	case CamelCase:
		structField.name = toLowerCamelCase(structField.name)
	default:
		structField.name = toSnakeCase(structField.name)
	}

	var tagValue string
	if field.Tag != nil {
		tagValue = field.Tag.Value
	}
	structTag := reflect.StructTag(strings.Replace(tagValue, "`", "", -1))
	// `json:"tag"` -> json:"tag"
	jsonTag := structTag.Get("json")
	// json:"tag,hoge"
	if strings.Contains(jsonTag, ",") {
		// json:",hoge"
		if strings.HasPrefix(jsonTag, ",") {
			jsonTag = ""
		} else {
			jsonTag = strings.SplitN(jsonTag, ",", 2)[0]
		}
	}
	if jsonTag == "-" {
		structField.name = ""
	} else if jsonTag != "" {
		structField.name = jsonTag
	}

	var tagDataMap = make(map[string]string)
	var fieldTagOld string
	if field.Doc != nil {
		for _, comment := range field.Doc.List {
			if comment != nil {
				fieldTag, tag := tagFromComment(comment.Text)
				switch {
				case (fieldTag == "description") || (fieldTag == "" && fieldTagOld == "description"):
					fieldTag = "description"
					if oldDes, ok := tagDataMap[fieldTag]; ok {
						tagDataMap[fieldTag] = oldDes + "\n" + tag
					} else {
						tagDataMap[fieldTag] = tag
					}
				case fieldTag != "" && tag != "":
					tagDataMap[fieldTag] = tag
				}
				fieldTagOld = fieldTag
			}
		}
	}

	for fieldTag, tag := range tagDataMap {
		if tag != "" && fieldTag != "" && structTag.Get(fieldTag) == "" {
			tagValue = tagValue + " " + fieldTag + ":\"" + tag + "\""
		}
	}

	if tagValue == "" {
		return structField
	}

	if des, _ := tagDataMap["description"]; des != "" {
		structField.description = strings.Replace(des, "\\n", "\n", -1)
	}

	if typeTag, _ := tagDataMap["swaggertype"]; typeTag != "" {
		parts := strings.Split(typeTag, ",")
		if 0 < len(parts) && len(parts) <= 2 {
			newSchemaType := parts[0]
			newArrayType := structField.arrayType
			if len(parts) >= 2 {
				if newSchemaType == "array" {
					newArrayType = parts[1]
				} else if newSchemaType == "primitive" {
					newSchemaType = parts[1]
					newArrayType = parts[1]
				}
			}
			if newSchemaType == "datetime" {
				structField.schemaType = "string"
				structField.formatType = "date-time"
				structField.arrayType = newArrayType
			} else {
				CheckSchemaType(newSchemaType)
				CheckSchemaType(newArrayType)
				structField.schemaType = newSchemaType
				structField.arrayType = newArrayType
			}
		}
	}
	if hiddenTag, _ := tagDataMap["hidden"]; hiddenTag != "" {
		return nil
	}
	if exampleTag, _ := tagDataMap["example"]; exampleTag != "" {
		if ex := defineTypeOfExample(structField.schemaType, structField.arrayType, exampleTag); ex != nil {
			structField.exampleValue = ex
		}
	}
	if formatTag, _ := tagDataMap["format"]; formatTag != "" {
		structField.formatType = formatTag
	}
	if bindingTag, _ := tagDataMap["binding"]; bindingTag != "" {
		for _, val := range strings.Split(bindingTag, ",") {
			if val == "required" {
				structField.isRequired = true
				break
			}
		}
	}
	if validateTag, _ := tagDataMap["validate"]; validateTag != "" {
		for _, val := range strings.Split(validateTag, ",") {
			if val == "required" {
				structField.isRequired = true
				break
			}
		}
	}
	if extensionsTag, _ := tagDataMap["extensions"]; extensionsTag != "" {
		structField.extensions = map[string]interface{}{}
		for _, val := range strings.Split(extensionsTag, ",") {
			parts := strings.SplitN(val, "=", 2)
			if len(parts) == 2 {
				structField.extensions[parts[0]] = parts[1]
			} else {
				structField.extensions[parts[0]] = true
			}
		}
	}
	if enumsTag, _ := tagDataMap["enums"]; enumsTag != "" {
		enumType := structField.schemaType
		if structField.schemaType == "array" {
			enumType = structField.arrayType
		}

		for _, e := range strings.Split(enumsTag, ",") {
			structField.enums = append(structField.enums, defineType(enumType, e))
		}
	}
	if defaultTag, _ := tagDataMap["default"]; defaultTag != "" {
		structField.defaultValue = defineType(structField.schemaType, defaultTag)
	}

	if IsNumericType(structField.schemaType) || IsNumericType(structField.arrayType) {
		structField.maximum = getFloatTag(tagDataMap, "maximum")
		structField.minimum = getFloatTag(tagDataMap, "minimum")
	}
	if structField.schemaType == "string" || structField.arrayType == "string" {
		structField.maxLength = getIntTag(tagDataMap, "maxLength")
		structField.minLength = getIntTag(tagDataMap, "minLength")
	}

	return structField
}

func (parser *Parser) parseStruct(path string, field *ast.Field, astFile *ast.File) (properties map[string]*genswagger.Type) {
	properties = map[string]*genswagger.Type{}
	structField := parser.parseField(field, astFile, path)
	if structField == nil || structField.name == "" {
		return
	}
	var desc = structField.description

	if structField.path != "" {
		path = structField.path
	}
	if _, ok := parser.fullPathDefinitions[path][structField.schemaType]; ok { // user type field
		// write definition if not yet present
		refTypeName := parser.pathAndName(path, structField.schemaType)
		if _, ok := parser.schemaParsedDefinitions[refTypeName]; !ok {
			parser.ParseSchema(path, structField.schemaType)
		}
		properties[structField.name] = &genswagger.Type{
			Description: desc,
			Ref:         "#/definitions/" + refTypeName,
		}
	} else if structField.schemaType == "array" { // array field type
		// if defined -- ref it
		if _, ok := parser.fullPathDefinitions[path][structField.arrayType]; ok { // user type in array\
			refTypeName := parser.pathAndName(path, structField.arrayType)
			if _, ok := parser.schemaParsedDefinitions[refTypeName]; !ok {
				parser.ParseSchema(path, structField.arrayType)
			}
			properties[structField.name] = &genswagger.Type{
				Type:        structField.schemaType,
				Description: desc,
				Items: &genswagger.Type{
					Ref: "#/definitions/" + refTypeName,
				},
			}
		} else { // standard type in array
			required := make([]string, 0)
			if structField.isRequired {
				required = append(required, structField.name)
			}

			properties[structField.name] = &genswagger.Type{

				Type:        structField.schemaType,
				Description: desc,
				Format:      structField.formatType,
				Required:    required,
				Items: &genswagger.Type{
					Type:      structField.arrayType,
					Maximum:   floatPointerToV(structField.maximum),
					Minimum:   floatPointerToV(structField.minimum),
					MaxLength: intPointerToV(structField.maxLength),
					MinLength: intPointerToV(structField.minLength),
					Enum:      structField.enums,
					Default:   structField.defaultValue,
				},
				Example: structField.exampleValue,
			}
		}
	} else {
		required := make([]string, 0)
		if structField.isRequired {
			required = append(required, structField.name)
		}
		properties[structField.name] = &genswagger.Type{
			Type:        structField.schemaType,
			Description: desc,
			Format:      structField.formatType,
			Required:    required,
			Maximum:     floatPointerToV(structField.maximum),
			Minimum:     floatPointerToV(structField.minimum),
			MaxLength:   intPointerToV(structField.maxLength),
			MinLength:   intPointerToV(structField.minLength),
			Enum:        structField.enums,
			Default:     structField.defaultValue,
			Example:     structField.exampleValue,
			Extensions:  structField.extensions,
		}
		nestStruct, ok := field.Type.(*ast.StructType)
		if ok {
			props := map[string]*genswagger.Type{}
			nestRequired := make([]string, 0)
			for _, v := range nestStruct.Fields.List {
				p := parser.parseStruct(path, v, astFile)
				for k, v := range p {
					if v.Type != "object" {
						nestRequired = append(nestRequired, v.Required...)
						v.Required = make([]string, 0)
					}
					props[k] = v
				}
			}

			properties[structField.name] = &genswagger.Type{
				Type:        structField.schemaType,
				Description: desc,
				Format:      structField.formatType,
				Properties:  props,
				Required:    nestRequired,
				Maximum:     floatPointerToV(structField.maximum),
				Minimum:     floatPointerToV(structField.minimum),
				MaxLength:   intPointerToV(structField.maxLength),
				MinLength:   intPointerToV(structField.minLength),
				Enum:        structField.enums,
				Default:     structField.defaultValue,
				Example:     structField.exampleValue,
			}
		}
	}
	return
}

func (parser *Parser) collectRequiredFields(path string, properties map[string]*genswagger.Type, extraRequired []string) (requiredFields []string) {
	// created sorted list of properties keys so when we iterate over them it's deterministic
	ks := make([]string, 0, len(properties))
	for k := range properties {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	requiredFields = make([]string, 0)

	// iterate over keys list instead of map to avoid the random shuffle of the order that go does for maps
	for _, k := range ks {
		prop := properties[k]

		// todo find the pkgName of the property type
		tname := prop.Type
		if _, ok := parser.fullPathDefinitions[path][tname]; ok {
			//tspec := parser.fullPathDefinitions[path][tname]
			if _, ok := parser.schemaParsedDefinitions[parser.pathAndName(path, tname)]; !ok {
				parser.ParseSchema(path, tname)
			}
		}
		if tname != "object" {
			requiredFields = append(requiredFields, prop.Required...)
		}
		properties[k] = prop
	}

	if extraRequired != nil {
		requiredFields = append(requiredFields, extraRequired...)
	}

	sort.Strings(requiredFields)

	return
}

// check and compute level clean name of full name schema
func (parser *Parser) computeLevelNameOfDefinition() {
	for name := range parser.Definitions {
		if _, ok := parser.levelName[name]; !ok {
			parser.levelName[name] = 0
		}
		for i := 0; i < 4; i++ {
			nameUse := getNameByLevel(name, parser.levelName[name])
			if _, ok := parser.countUseName[nameUse]; ok {
				parser.countUseName[nameUse] = 2
				parser.levelName[name] = parser.levelName[name] + 1
				if _, ok := parser.trueNameDefinitionReverse[nameUse]; ok {
					for _, nameShouldUpdate := range parser.trueNameDefinitionReverse[nameUse] {
						parser.levelName[nameShouldUpdate] += 1
					}
					delete(parser.trueNameDefinitionReverse, nameUse)
				}
			} else {
				if parser.trueNameDefinitionReverse[nameUse] == nil {
					parser.trueNameDefinitionReverse[nameUse] = []string{name}
				} else {
					parser.trueNameDefinitionReverse[nameUse] = append(parser.trueNameDefinitionReverse[nameUse], name)
				}
				parser.countUseName[nameUse] = 1
				break
			}
		}
	}
}

func writeToFileTest(b []byte, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(b)
	return err
}

// inject Definitions of parser to swagger of parser after change computeLevelNameOfDefinition
func (parser *Parser) mergeDefinitionsToSwagger() {
	parser.computeLevelNameOfDefinition()
	for name := range parser.Definitions {
		var useName string
		if level, ok := parser.levelName[name]; ok {
			useName = getNameByLevel(name, level)
		} else {
			useName = getNameByLevel(name, 0)
		}
		parser.trueNameDefinition[name] = useName
	}
	for name, swType := range parser.Definitions {
		parser.swagger.Definitions[parser.fixKeyRef(name)] = parser.fixSwaggerType(swType)
	}
}

// name change #ref of Type after change path of definition
func (parser *Parser) fixSwaggerType(swType *genswagger.Type) *genswagger.Type {
	if swType == nil {
		return nil
	}
	if swType.Ref != "" {
		swType.Type = ""
		swType.Ref = parser.fixRef(swType.Ref)
		if swType.Ref == "#/definitions/" {
			swType.Ref = ""
			swType.Type = "object"
		}
	}
	if swType.Items != nil {
		swType.Items = parser.fixSwaggerType(swType.Items)
	}
	if swType.Properties != nil && len(swType.Properties) > 0 {
		tmp := make(map[string]*genswagger.Type)
		for i := range swType.Properties {
			tmp[i] = parser.fixSwaggerType(swType.Properties[i])
		}
		swType.Properties = tmp
	}
	if swType.Dependencies != nil && len(swType.Dependencies) > 0 {
		tmp := make(map[string]*genswagger.Type)
		for i := range swType.Dependencies {
			tmp[i] = parser.fixSwaggerType(swType.Dependencies[i])
		}
		swType.Dependencies = tmp
	}

	if swType.AdditionalItems != nil {
		swType.AdditionalItems = parser.fixSwaggerType(swType.AdditionalItems)
	}
	if swType.PatternProperties != nil {
		swType.PatternProperties = parser.fixSwaggerType(swType.PatternProperties)
	}
	if swType.AllOf != nil && len(swType.AllOf) > 0 {
		tmp := make([]*genswagger.Type, 0)
		for i := range swType.AllOf {
			tmp = append(tmp, parser.fixSwaggerType(swType.AllOf[i]))
		}
		swType.AllOf = tmp
	}

	if swType.AnyOf != nil && len(swType.AnyOf) > 0 {
		tmp := make([]*genswagger.Type, 0)
		for i := range swType.AnyOf {
			tmp = append(tmp, parser.fixSwaggerType(swType.AnyOf[i]))
		}
		swType.AnyOf = tmp
	}
	return swType
}

func (parser *Parser) fixRef(ref string) string {
	if !strings.Contains(ref, "#/definitions/") {
		return ref
	}
	ref = strings.Replace(ref, "#/definitions/", "", 1)
	if trueName, ok := parser.trueNameDefinition[ref]; ok {
		ref = trueName
	}
	return "#/definitions/" + ref
}

func (parser *Parser) fixKeyRef(keyRef string) string {
	if trueName, ok := parser.trueNameDefinition[keyRef]; ok {
		keyRef = trueName
	}
	return keyRef
}

func getNameByLevel(name string, level int) string {
	switch level {
	case 0:
		namePath := strings.Split(name, ".")
		return namePath[len(namePath)-1]
	case 1:
		namePath := strings.Split(name, ".")
		if len(namePath) == 1 {
			return name
		}
		name := namePath[len(namePath)-1]
		path := namePath[0]
		pathSplit := strings.Split(path, "/")
		return pathSplit[len(pathSplit)-1] + "." + name
	case 2:
		namePath := strings.Split(name, ".")
		if len(namePath) == 1 {
			return name
		}
		name := namePath[len(namePath)-1]
		path := namePath[0]
		pathSplit := strings.Split(path, "/")
		if len(pathSplit) == 1 {
			return pathSplit[len(pathSplit)-1] + "." + name
		}
		return fmt.Sprintf("%s_%s.%s", pathSplit[len(pathSplit)-2], pathSplit[len(pathSplit)-1], name)
	default:
		return strings.Replace(name, "/", "_", -1)
	}
}

var tagSupports = map[string]bool{"required": true, "description": true,
	"example": true, "enums": true, "title": true, "dedicated": true, "default": true,
	"swaggertype": true, "format": true, "binding": true, "validate": true,
	"extensions": true, "maximum": true, "minimum": true, "maxlength": true, "minlength": true, "type": true, "hidden": true}

var rBeeComment = regexp.MustCompile(`^//\s*@([a-zA-Z0-9_]+)[:]*\s*(.*)$`)

func tagFromComment(comment string) (field, tag string) {
	commentTrim := strings.Trim(comment, ",: \"")
	if commentTrim == "Dedicated" {
		return "dedicated", `dedicated:"true"`
	}
	if allCommentClean := strings.Trim(commentTrim, " /"); len(allCommentClean) > 0 && allCommentClean[0] != '@' {
		tmp := strings.Trim(commentTrim, "/")
		if tmp[0] == ' ' {
			tmp = tmp[1:]
		}
		return "", tmp
	}
	match := rBeeComment.FindStringSubmatch(comment)
	if len(match) == 3 {
		match[1] = strings.ToLower(match[1])
		if match[1] == "required" {
			return "binding", `binding:"required"`
		}
		if match[1] == "dedicated" {
			return "dedicated", `dedicated:"true"`
		}
		if match[1] == "hidden" {
			return "hidden", `hidden:"true"`
		}
		if match[1] == "valid" {
			match[1] = "enums"
		}
		if _, ok := tagSupports[match[1]]; !ok {
			fmt.Println("Warn:", match[1], "does not support")
		}
		if match[1] == "type" {
			match[1] = "swaggertype"
		}
		return match[1], strings.Replace(strings.Trim(match[2], ",: \""), "\"", "", -1)

		//return match[1], match[1] + ":\"" + strings.Replace(strings.Trim(match[2], ",: \""), "\"", "", -1) + "\""
	}
	return
}

func GetNamespaceFromNamespaceComment(commentLine string) []string {
	re := regexp.MustCompile("[\\s\\t]+")
	return re.Split(commentLine, -1)
}
