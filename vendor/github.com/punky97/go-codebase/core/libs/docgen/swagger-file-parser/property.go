package swag

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"
)

// ErrFailedConvertPrimitiveType Failed to convert for swag to interpretable type
var ErrFailedConvertPrimitiveType = errors.New("swag property: failed convert primitive type")

type propertyName struct {
	SchemaType string
	ArrayType  string
	CrossPkg   string
}

type propertyNewFunc func(schemeType string, crossPkg string) propertyName

func newArrayProperty(schemeType string, crossPkg string) propertyName {
	return propertyName{
		SchemaType: "array",
		ArrayType:  schemeType,
		CrossPkg:   crossPkg,
	}
}

func newProperty(schemeType string, crossPkg string) propertyName {
	return propertyName{
		SchemaType: schemeType,
		ArrayType:  "string",
		CrossPkg:   crossPkg,
	}
}

func convertFromSpecificToPrimitive(typeName string) (string, error) {
	typeName = strings.ToUpper(typeName)
	switch typeName {
	case "TIME", "OBJECTID", "UUID":
		return "string", nil
	case "DECIMAL":
		return "number", nil
	}
	return "", ErrFailedConvertPrimitiveType
}

func parseFieldSelectorExpr(astFile *ast.File, filePath string,
	astTypeSelectorExpr *ast.SelectorExpr, parser *Parser, propertyNewFunc propertyNewFunc) propertyName {
	if primitiveType, err := convertFromSpecificToPrimitive(astTypeSelectorExpr.Sel.Name); err == nil {
		return propertyNewFunc(primitiveType, "")
	}

	if pkgName, ok := astTypeSelectorExpr.X.(*ast.Ident); ok {
		path, schemaName := parser.getPathOfSchemaShouldParse(filePath, pkgName.Name+"."+astTypeSelectorExpr.Sel.Name, astFile)
		fullPath := parser.pathAndName(path, schemaName)
		if _, ok := parser.schemaParsedDefinitions[fullPath]; !ok {
			parser.ParseSchema(path, schemaName)
		}
		return propertyNewFunc(schemaName, path)
	}

	fmt.Printf("%s is not supported. but it will be set with string temporary. Please report any problems.\n", astTypeSelectorExpr.Sel.Name)
	return propertyName{SchemaType: "string", ArrayType: "string"}
}

// getPropertyName returns the string value for the given field if it exists, otherwise it panics.
// allowedValues: array, boolean, integer, null, number, object, string
func getPropertyName(astFile *ast.File, filePath string,
	field *ast.Field, parser *Parser) propertyName {

	// check if it is a custom type
	typeName := fmt.Sprintf("%v", field.Type)
	if fTXstar, ok := field.Type.(*ast.StarExpr); ok {
		if fTXstar.X != nil {
			if xIdent, canAssert := fTXstar.X.(*ast.Ident); canAssert {
				typeName = xIdent.Name
			}
		}
	} else if typeSelector, ok :=  field.Type.(*ast.SelectorExpr); ok {
		typeName = typeSelector.Sel.Name
	}
	if actualPrimitiveType, isCustomType := parser.CustomPrimitiveTypes[typeName]; isCustomType {
		return propertyName{SchemaType: actualPrimitiveType, ArrayType: actualPrimitiveType}
	}

	if astTypeSelectorExpr, ok := field.Type.(*ast.SelectorExpr); ok {
		return parseFieldSelectorExpr(astFile, filePath, astTypeSelectorExpr, parser, newProperty)
	}

	if astTypeIdent, ok := field.Type.(*ast.Ident); ok {
		name := astTypeIdent.Name
		schemeType := TransToValidSchemeType(name)
		return propertyName{SchemaType: schemeType, ArrayType: schemeType}
	}
	if ptr, ok := field.Type.(*ast.StarExpr); ok {
		if astTypeSelectorExpr, ok := ptr.X.(*ast.SelectorExpr); ok {
			return parseFieldSelectorExpr(astFile, filePath, astTypeSelectorExpr, parser, newProperty)
		}
		// TODO support custom pointer type?
		if _, ok := ptr.X.(*ast.MapType); ok { // if map
			//TODO support map
			return propertyName{SchemaType: "object", ArrayType: "object"}
		}
		if _, ok := ptr.X.(*ast.StructType); ok { // if struct
			return propertyName{SchemaType: "object", ArrayType: "object"}
		}
		if astTypeIdent, ok := ptr.X.(*ast.Ident); ok {
			name := astTypeIdent.Name
			schemeType := TransToValidSchemeType(name)
			return propertyName{SchemaType: schemeType, ArrayType: schemeType}
		}
		if astTypeArray, ok := ptr.X.(*ast.ArrayType); ok { // if array
			if astTypeArrayExpr, ok := astTypeArray.Elt.(*ast.SelectorExpr); ok {
				return parseFieldSelectorExpr(astFile, filePath, astTypeArrayExpr, parser, newArrayProperty)
			}
			if astTypeArrayIdent, ok := astTypeArray.Elt.(*ast.Ident); ok {
				name := TransToValidSchemeType(astTypeArrayIdent.Name)
				return propertyName{SchemaType: "array", ArrayType: name}
			}
		}
	}
	if astTypeArray, ok := field.Type.(*ast.ArrayType); ok { // if array
		if astTypeArrayExpr, ok := astTypeArray.Elt.(*ast.SelectorExpr); ok {
			return parseFieldSelectorExpr(astFile, filePath, astTypeArrayExpr, parser, newArrayProperty)
		}
		if astTypeArrayExpr, ok := astTypeArray.Elt.(*ast.StarExpr); ok {
			if astTypeArraySel, ok := astTypeArrayExpr.X.(*ast.SelectorExpr); ok {
				return parseFieldSelectorExpr(astFile, filePath, astTypeArraySel, parser, newArrayProperty)
			}
			if astTypeArrayIdent, ok := astTypeArrayExpr.X.(*ast.Ident); ok {
				name := TransToValidSchemeType(astTypeArrayIdent.Name)
				return propertyName{SchemaType: "array", ArrayType: name}
			}
		}
		itemTypeName := TransToValidSchemeType(fmt.Sprintf("%s", astTypeArray.Elt))
		if strings.Contains(itemTypeName, "&{%!s") {
			return propertyName{SchemaType: "array", ArrayType: "object"}
		}
		if actualPrimitiveType, isCustomType := parser.CustomPrimitiveTypes[itemTypeName]; isCustomType {
			itemTypeName = actualPrimitiveType
		}
		return propertyName{SchemaType: "array", ArrayType: itemTypeName}
	}
	if _, ok := field.Type.(*ast.MapType); ok { // if map
		//TODO: support map
		return propertyName{SchemaType: "object", ArrayType: "object"}
	}
	if _, ok := field.Type.(*ast.StructType); ok { // if struct
		return propertyName{SchemaType: "object", ArrayType: "object"}
	}
	if _, ok := field.Type.(*ast.InterfaceType); ok { // if interface{}
		return propertyName{SchemaType: "object", ArrayType: "object"}
	}
	panic("not supported" + fmt.Sprint(field.Type))
}
