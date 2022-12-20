package swag

import (
	"github.com/punky97/go-codebase/core/libs/docgen/genswagger"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type PathInfo struct {
	Path string
	Tags []string
}

// ParseAPI parses general api info for give searchDir (path of cmd) and mainAPIFile (main api path)
func (p *Parser) ParseMultiAPI(paths []PathInfo, codeNamePaths []string) (*genswagger.SwaggerObject, error) {
	for _, codeNamePath := range codeNamePaths {
		p.getCodename(codeNamePath)
	}
	for _, path := range paths {
		p.pathParsed[path.Path] = true
		if err := p.getAllGoFileInfo(path.Path); err != nil {
			continue
		}
		codenameInfo := p.parseBasePathFromMain(path.Path + "/main.go")
		if codenameInfo == nil {
			continue
		}
		codenameInfo.tags = path.Tags
		webserverPath := p.getWebserverPath(path.Path)
		if webserverPath == "" {
			continue
		}
		handlerUse := p.ParseInitApi(webserverPath, *codenameInfo)
		if handlerUse == nil || len(handlerUse.handlers) < 1 {
			fmt.Println("WARN -  not found any route from webserverPath:", webserverPath)
			continue
		}
		for pkgName := range handlerUse.handlers {
			for _, importPath := range handlerUse.imports {
				path := strings.Trim(importPath.Path.Value, `" `)
				if pathImportIgnore(path) {
					continue
				}
				if importPath.Name != nil || getEndOfPath(path) == pkgName || p.checkSamePkgAndImport(path, pkgName) {
					if err := p.getAllGoMultipleFileInfo(path, codenameInfo, pkgName); err != nil {
						fmt.Println("ERROR - catch error when getAllGoMultipleFileInfo of path:", path, "- err: ", err)
						continue
					}
				}
			}
		}

	}

	for codename, files := range p.multiFiles {
		for fileName, astFile := range files {
			p.GetDefNeedParseFromRouterAPI(fileName, astFile, codename)
		}
	}
	p.mergeDefinitionsToSwagger()

	for codenameInfo, files := range p.multiFiles {
		for fileName, astFile := range files {
			if err := p.ParseRouterAPIInfo(fileName, astFile, codenameInfo); err != nil {
				return nil, err
			}
		}
	}
	return p.swagger, nil
}

// GetAllGoFileInfo gets all Go source files information for given searchDir.
func (p *Parser) getAllGoMultipleFileInfo(searchDir string, codename *parseCodenameInfo, pkgImportName string) error {
	path := p.basePath + searchDir
	files, _ := p.getFilesFromPath(path)
	if _, ok := p.filePathPkgStore[codename.name]; !ok {
		p.filePathPkgStore[codename.name] = make(map[string]string)
	}
	p.filePathPkgStore[codename.name][path] = pkgImportName
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".go" {
			fset := token.NewFileSet() // positions are relative to fset
			fileGoPath := path + "/" + file.Name()
			astFile, err := parser.ParseFile(fset, fileGoPath, nil, parser.ParseComments)
			if err != nil {
				fmt.Println("ParseFile error:", err)
				continue
			}

			if files, ok := p.multiFiles[codename]; ok {
				files[fileGoPath] = astFile
				p.multiFiles[codename] = files
			} else {
				p.multiFiles[codename] = map[string]*ast.File{fileGoPath: astFile}
			}
		}
	}
	return nil
	//return filepath.Walk(p.basePath + searchDir, p.visitWithCodeName)
}

func (p *Parser) visitWithCodeName(path string, f os.FileInfo, err error) error {
	if err := Skip(f); err != nil {
		return err
	}

	if ext := filepath.Ext(path); ext == ".go" {
		fset := token.NewFileSet() // positions are relative to fset
		astFile, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("ParseFile error:%+v", err)
		}
		if p.currentCodename == nil {
			return nil
		}
		if files, ok := p.multiFiles[p.currentCodename]; ok {
			files[path] = astFile
			p.multiFiles[p.currentCodename] = files
		} else {
			p.multiFiles[p.currentCodename] = map[string]*ast.File{path: astFile}
		}
	}
	return nil
}

func (p *Parser) getWebserverPath(path string) string {
	if _, err := os.Stat(path + "/app/webserver.go"); os.IsNotExist(err) {

	}
	return path + "/app/webserver.go"

}

func (p *Parser) getCodename(codeNamePath string) {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, codeNamePath, nil, parser.ParseComments)
	if err != nil {
		return
	}
	for _, decl := range astFile.Decls {
		if genDecl, gdAssert := decl.(*ast.GenDecl); gdAssert {
			p.getBasePathAll(genDecl)
		}
	}
}

func (p *Parser) getBasePathAll(gen *ast.GenDecl) {
	if gen == nil || gen.Specs == nil {
		return
	}
	for _, spec := range gen.Specs {
		name, info := p.getBasePath(spec)
		if name != "" {
			p.CodeName[name] = info
		}
	}
	return
}

func (p *Parser) getBasePath(spec ast.Spec) (apiName string, serviceInfo parseCodenameInfo) {
	if valueSpec, vsAssert := spec.(*ast.ValueSpec); vsAssert {
		if len(valueSpec.Names) == 1 && len(valueSpec.Values) == 1 {
			name := valueSpec.Names[0]
			if name.Obj == nil || name.Obj.Decl == nil {
				return "", parseCodenameInfo{}
			}
			apiName = name.Name
			if valueCompose, vcAssert := valueSpec.Values[0].(*ast.CompositeLit); vcAssert {
				for _, elt := range valueCompose.Elts {
					if eltKeyValue, ekvAssert := elt.(*ast.KeyValueExpr); ekvAssert {
						if eltKeyValue.Value == nil || eltKeyValue.Key == nil {
							continue
						}
						if keyIdent, kiAssert := eltKeyValue.Key.(*ast.Ident); kiAssert && keyIdent != nil {
							switch keyIdent.Name {
							case "HTTPBasePath":
								if valueBasicLit, vblAssert := eltKeyValue.Value.(*ast.BasicLit); vblAssert {
									tmp := strings.Trim(valueBasicLit.Value, "\"")
									if tmp != "" {
										serviceInfo.basePath = strings.Trim(valueBasicLit.Value, "\"")
									}
								}
							case "CodeName":
								if valueBasicLit, vblAssert := eltKeyValue.Value.(*ast.BasicLit); vblAssert {
									serviceInfo.name = strings.Trim(valueBasicLit.Value, "\"")
								}
							}
						}
					}
				}
			}
		}
	}
	return

}

func (p *Parser) parseBasePathFromMain(mainPath string) *parseCodenameInfo {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, mainPath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}
	for _, decl := range astFile.Decls {
		if declFun, dfAssert := decl.(*ast.FuncDecl); dfAssert {
			if declFun.Name != nil && declFun.Name.Name == "main" && declFun.Body != nil && declFun.Body.List != nil {
				for _, stmt := range declFun.Body.List {
					if deferStmt, ldfAssert := stmt.(*ast.DeferStmt); ldfAssert {
						if deferStmt.Call != nil && deferStmt.Call.Args != nil {
							for _, arg := range deferStmt.Call.Args {
								if argIdent, aiAssert := arg.(*ast.Ident); aiAssert {
									if argIdent.Obj != nil && argIdent.Obj.Decl != nil {
										if argAssign, aaAssert := argIdent.Obj.Decl.(*ast.AssignStmt); aaAssert {
											if argAssign.Rhs != nil {
												for _, rh := range argAssign.Rhs {
													if rhCallExpr, rhceAssert := rh.(*ast.CallExpr); rhceAssert {
														if rhCallExpr.Args != nil {
															for _, rhArg := range rhCallExpr.Args {
																if rhSelector, rhsAssert := rhArg.(*ast.SelectorExpr); rhsAssert {
																	if rhSelector.X != nil && rhSelector.Sel != nil && rhSelector.Sel.Name == "HTTPBasePath" {
																		if rhxSelector, rhxAssert := rhSelector.X.(*ast.SelectorExpr); rhxAssert {
																			if rhxSelector.Sel != nil {
																				if codeName, ok := p.CodeName[rhxSelector.Sel.Name]; ok {
																					return &codeName
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
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (p *Parser) ParseInitApi(webserverPath string, codenameInfo parseCodenameInfo) *handlerUse {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, webserverPath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}
	handlerUse := handlerUse{
		imports:  []ast.ImportSpec{},
		handlers: make(map[string]string),
	}
	for _, decl := range astFile.Decls {
		switch funcDecl := decl.(type) {
		case *ast.FuncDecl:
			if !CheckReturn(funcDecl.Type, "transhttp", "Routes") {
				continue
			}
			if funcDecl.Body != nil && funcDecl.Body.List != nil && len(funcDecl.Body.List) > 0 {
				for _, stmt := range funcDecl.Body.List {
					if rs, ok := stmt.(*ast.ReturnStmt); ok {
						if rs.Results != nil && len(rs.Results) > 0 {
							for _, re := range rs.Results {
								if compositeLit, ok := re.(*ast.CompositeLit); ok {
									if compositeLit.Elts != nil && compositeLit.Type != nil {
										for _, exlt := range compositeLit.Elts {
											if exltComposLit, canDDD := exlt.(*ast.CompositeLit); canDDD {
												routeInfo := getInfoRoute(exltComposLit.Elts)
												routeInfo.Pattern = codenameInfo.basePath + routeInfo.Pattern
												if codenameInfo.name != "" {
													handlerUse.handlers[routeInfo.HandlerPkgName] = routeInfo.HandlerName
													p.HandlerName[fmt.Sprintf("%s.%s.%s", codenameInfo.name, routeInfo.HandlerPkgName, routeInfo.HandlerName)] = routeInfo
													//p.HandlerName[codenameInfo.name + "." + routeInfo.HandlerName] = routeInfo
												} else {
													p.HandlerName[routeInfo.HandlerName] = routeInfo
												}
											}
										}
									} else {
										break
									}
								} else {
									break
								}
							}
						}
					}
				}
			}
		}
	}
	if len(handlerUse.handlers) > 0 {
		var imports = make([]ast.ImportSpec, 0)
		for _, importSpec := range astFile.Imports {
			if importSpec != nil {
				imports = append(imports, *importSpec)
			}
		}
		handlerUse.imports = imports
		return &handlerUse
	}
	return nil
}

func CheckReturn(funcType *ast.FuncType, xName string, selName string) bool {
	if funcType == nil {
		return false
	}
	returns := funcType.Results
	if returns == nil || returns.List == nil || len(returns.List) != 1 || returns.List[0].Type == nil {
		return false
	}
	if selectorExpr, ok := returns.List[0].Type.(*ast.SelectorExpr); ok {
		if selectorExpr.X == nil || selectorExpr.Sel == nil {
			return false
		}
		if xIdent, ok := selectorExpr.X.(*ast.Ident); ok {
			return xIdent.Name == xName && selectorExpr.Sel.Name == selName
		}
	}
	//http.Me
	return false
}

func getInfoRoute(elts []ast.Expr) (res genswagger.SwaggerRouteInfo) {
	for _, elt := range elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if keyIdent, keyCanAssert := kv.Key.(*ast.Ident); keyCanAssert {
				if kv.Value == nil {
					continue
				}
				switch keyIdent.Name {
				case "Name":
					if valueIdent, valCanAssert := kv.Value.(*ast.BasicLit); valCanAssert {
						res.Name = strings.Trim(valueIdent.Value, "\"")
					}
				case "Method":
					if valueIdent, valCanAssert := kv.Value.(*ast.SelectorExpr); valCanAssert {
						if valueIdent.Sel != nil {
							switch valueIdent.Sel.Name {
							case "MethodGet":
								res.Method = http.MethodGet
							case "MethodPost":
								res.Method = http.MethodPost
							case "MethodPut":
								res.Method = http.MethodPut
							case "MethodDelete":
								res.Method = http.MethodDelete
							case "MethodPatch":
								res.Method = http.MethodPatch
							}
						}
					}
				case "Pattern":
					if valueIdent, valCanAssert := kv.Value.(*ast.BasicLit); valCanAssert {
						res.Pattern = strings.Trim(valueIdent.Value, "\"")
					}
				case "Handler":
					if valueUnaryExpr, valUnaryExprCanAssert := kv.Value.(*ast.UnaryExpr); valUnaryExprCanAssert {
						if valueUnaryExpr.X != nil {
							if xHanIdent, canXHanIdent := valueUnaryExpr.X.(*ast.CompositeLit); canXHanIdent {
								if xHanIdent.Type != nil {
									if xHanTypeSe, xHanIdentSeAssert := xHanIdent.Type.(*ast.SelectorExpr); xHanIdentSeAssert {
										res.HandlerName = xHanTypeSe.Sel.Name
										res.HandlerPkgName = getNameIdent(xHanTypeSe.X)
										//handlerName = xHanTypeSe.Sel.Name
									}

								}
							}
						}
					}
				case "AuthInfo":
					if valueIdent, valCanAssert := kv.Value.(*ast.CompositeLit); valCanAssert {
						if valueIdent.Elts != nil {
							for _, elt := range valueIdent.Elts {
								if authKy, canKyAssert := elt.(*ast.KeyValueExpr); canKyAssert {
									if authKey, authKeyCanAssert := authKy.Key.(*ast.Ident); authKeyCanAssert {
										if kv.Value == nil {
											continue
										}
										switch authKey.Name {
										case "TokenType":
											if authValue, canAuthValueAssert := authKy.Value.(*ast.CallExpr); canAuthValueAssert {
												if authValue.Fun != nil {
													if authValueFunSelector, canAuthValueFunSelectorAssert := authValue.Fun.(*ast.SelectorExpr);
														canAuthValueFunSelectorAssert {
														if authValueFunSelector.X != nil {
															if xAuthValueFunSelector, canXAuthValueFunSelector :=
																authValueFunSelector.X.(*ast.SelectorExpr); canXAuthValueFunSelector {
																res.AuthInfo.TokenType = strings.Replace(xAuthValueFunSelector.Sel.Name, "TokenType_", "", 1)
															}
														}

													}
												}
											}
										case "RestrictScopes":
											if scopeAuthValuesComposite, canValScopeAuthAssert := authKy.Value.(*ast.CompositeLit);
												canValScopeAuthAssert {
												if scopeAuthValuesComposite.Elts != nil {
													for _, scopeElt := range scopeAuthValuesComposite.Elts {
														if scopeBasicLit, canScopeBasicLitAssert := scopeElt.(*ast.BasicLit); canScopeBasicLitAssert {
															if res.AuthInfo.RestrictScopes == nil {
																res.AuthInfo.RestrictScopes = []string{strings.Trim(scopeBasicLit.Value, "\"")}
															} else {
																res.AuthInfo.RestrictScopes = append(res.AuthInfo.RestrictScopes, strings.Trim(scopeBasicLit.Value, "\""))
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

				}

			}
		}
	}
	return res
}

func getNameIdent(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	if xIdent, ok := expr.(*ast.Ident); ok {
		return xIdent.Name
	}
	return ""
}