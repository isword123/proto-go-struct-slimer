package logic

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type ProtoGoParser struct {
	file *ast.File
	packageName string
	fileBaseName string
}

func (pp *ProtoGoParser)Parse(filePath string) bool {
	fs := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fs, filePath, nil, parser.ParseComments)

	if err != nil {
		log.Println("Parse proto go file failed", err)
		return false
	}

	pp.file = parsedFile

	pp.fileBaseName = strings.TrimSuffix(filepath.Base(filePath), ".pb.go")
	fmt.Println("file base name", filePath, pp.fileBaseName)

	pp.packageName = parsedFile.Name.Name + "_trans"

	return true
}

func (pp *ProtoGoParser)ParseDir(filePath string) bool {
	fs := token.NewFileSet()
	// parser.ParseDir()
	parsedFile, err := parser.ParseFile(fs, filePath, nil, parser.ParseComments)

	if err != nil {
		log.Println("Parse proto go file failed", err)
		return false
	}

	pp.file = parsedFile

	return true
}

func (pp *ProtoGoParser)getPackageName() string {
	if len(pp.packageName) == 0 {
		return "hello"
	}

	return pp.packageName
}

func (pp *ProtoGoParser) getFileBaseName() string {
	return pp.fileBaseName
}

func (pp *ProtoGoParser) GetStructsBytes() []byte {
	bufs := new(bytes.Buffer)

	bufs.WriteString(fmt.Sprintf("package %s\n\n", pp.getPackageName()))

	for _, decl := range pp.file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			tSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structExp, ok1 := tSpec.Type.(*ast.StructType)
			if !ok1 {
				continue
			}

			fmt.Println("struct", tSpec.Name)

			// 不是公开的数据结构，不处理
			if !ast.IsExported(tSpec.Name.Name) {
				continue
			}


			bufs.WriteString(fmt.Sprintf("type %s struct {\n", tSpec.Name))

			for _, field := range structExp.Fields.List {
				if len(field.Names) <= 0 {
					continue
				}

				fmt.Println("filed type---", reflect.TypeOf(field.Type), "field name ---", field.Names[0].Name)

				fieldName := field.Names[0].Name
				if strings.HasPrefix(fieldName, "XXX_") {
					continue
				}

				if ident, ok := field.Type.(*ast.Ident); ok {
					fmt.Println("names", field.Names, "type", ident.Name, "tag", field.Tag)
					bufs.WriteString(fmt.Sprintf("\t%s %s", field.Names[0].Name, ident.Name))
				} else if arrI, ok := field.Type.(*ast.ArrayType); ok {
					// *ast.ArrayType field name --- Titles
					if eleI, ok := arrI.Elt.(*ast.StarExpr); ok {
						if detailI, ok := eleI.X.(*ast.Ident); ok {
							bufs.WriteString(fmt.Sprintf("\t%s []*%s", field.Names[0].Name, detailI.Name))
						} else {
							fmt.Println("wrong identifier type", field.Names[0].Name, reflect.TypeOf(eleI.X))
						}
					} else if eleI, ok := arrI.Elt.(*ast.Ident); ok {
						bufs.WriteString(fmt.Sprintf("\t%s []%s", field.Names[0].Name, eleI.Name))
					} else {
						fmt.Println("wrong identifier type", field.Names[0].Name, reflect.TypeOf(arrI.Elt))
						continue
					}

				} else if starI, ok := field.Type.(*ast.StarExpr); ok {
					detailI, ok := starI.X.(*ast.Ident)
					if ok {
						bufs.WriteString(fmt.Sprintf("\t%s *%s", field.Names[0].Name, detailI.Name))
					} else {
						fmt.Println("Unknown star type", fieldName, starI.X)
					}
				} else {
					fmt.Println("Unknown type", fieldName, field.Type)
				}

				if field.Tag != nil {
					jsonTag, ok := pp.parseJSONTag(field.Tag.Value)
					if ok {
						bufs.WriteString(fmt.Sprintf(" `%s`\n", jsonTag))
					}
					fmt.Println("tag is", field.Tag.Value)
				} else {
					bufs.WriteString("\n")
				}
			}

			bufs.WriteString("}\n\n")
		}
	}

	return bufs.Bytes()
}

func (pp *ProtoGoParser) parseJSONTag(srcTag string) (string, bool) {
	index := strings.Index(srcTag, "json:")

	if index < 0 {
		return "", false
	}

	return srcTag[index:len(srcTag) - 1], true
}

func (pp *ProtoGoParser) saveNewCode(bs []byte, dir string) bool {
	fileName := filepath.Join(dir, fmt.Sprintf("%s.go", pp.getFileBaseName()))

	err := ioutil.WriteFile(fileName, bs, os.ModePerm)
	if err != nil {
		fmt.Println("Save new code failed", fileName, err.Error())
		return false
	}

	return true
}

func (pp *ProtoGoParser) ParseAndSave(filePath string, dir string) bool {
	pp.Parse(filePath)
	bs := pp.GetStructsBytes()
	return pp.saveNewCode(bs, dir)
}