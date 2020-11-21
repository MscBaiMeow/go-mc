// gen_blocks.go generates block information.

//+build ignore

package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"net/http"
	"os"
	"reflect"
	"strconv"

	"github.com/iancoleman/strcase"
)

const (
	infoURL = "https://raw.githubusercontent.com/PrismarineJS/minecraft-data/master/data/pc/1.16.2/blocks.json"
)

type Block struct {
	ID          uint32 `json:"id"`
	DisplayName string `json:"displayName"`
	Name        string `json:"name"`

	Hardness   float64         `json:"hardness"`
	Diggable   bool            `json:"diggable"`
	DropIDs    []uint32        `json:"drops"`
	NeedsTools map[uint32]bool `json:"harvestTools"`

	MinStateID uint32 `json:"minStateId"`
	MaxStateID uint32 `json:"maxStateId"`

	Transparent      bool `json:"transparent"`
	FilterLightLevel int  `json:"filterLight"`
	EmitLightLevel   int  `json:"emitLight"`
}

func downloadInfo() ([]Block, error) {
	resp, err := http.Get(infoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []Block
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func makeBlockDeclaration(blocks []Block) *ast.DeclStmt {
	out := &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR}}

	for _, b := range blocks {
		t := reflect.TypeOf(b)
		fields := make([]ast.Expr, t.NumField())

		for i := 0; i < t.NumField(); i++ {
			ft := t.Field(i)

			var val ast.Expr
			switch ft.Type.Kind() {
			case reflect.Uint32, reflect.Int:
				val = &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(reflect.ValueOf(b).Field(i))}
			case reflect.Float64:
				val = &ast.BasicLit{Kind: token.FLOAT, Value: fmt.Sprint(reflect.ValueOf(b).Field(i))}
			case reflect.String:
				val = &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(reflect.ValueOf(b).Field(i).String())}
			case reflect.Bool:
				val = &ast.BasicLit{Kind: token.IDENT, Value: fmt.Sprint(reflect.ValueOf(b).Field(i).Bool())}

			case reflect.Slice:
				val = &ast.CompositeLit{
					Type: &ast.ArrayType{
						Elt: &ast.BasicLit{Kind: token.IDENT, Value: ft.Type.Elem().Name()},
					},
				}
				v := reflect.ValueOf(b).Field(i)
				switch ft.Type.Elem().Kind() {
				case reflect.Uint32, reflect.Int:
					for x := 0; x < v.Len(); x++ {
						val.(*ast.CompositeLit).Elts = append(val.(*ast.CompositeLit).Elts, &ast.BasicLit{
							Kind:  token.INT,
							Value: fmt.Sprint(v.Index(x)),
						})
					}
				}

			case reflect.Map:
				// Must be the NeedsTools map of type map[uint32]bool.
				m := &ast.CompositeLit{
					Type: &ast.MapType{
						Key:   &ast.BasicLit{Kind: token.IDENT, Value: ft.Type.Key().Name()},
						Value: &ast.BasicLit{Kind: token.IDENT, Value: ft.Type.Elem().Name()},
					},
				}
				iter := reflect.ValueOf(b).Field(i).MapRange()
				for iter.Next() {
					m.Elts = append(m.Elts, &ast.KeyValueExpr{
						Key:   &ast.BasicLit{Kind: token.INT, Value: fmt.Sprint(iter.Key().Uint())},
						Value: &ast.BasicLit{Kind: token.IDENT, Value: fmt.Sprint(iter.Value().Bool())},
					})
				}

				val = m
			}

			fields[i] = &ast.KeyValueExpr{
				Key:   &ast.Ident{Name: ft.Name},
				Value: val,
			}
		}

		out.Decl.(*ast.GenDecl).Specs = append(out.Decl.(*ast.GenDecl).Specs, &ast.ValueSpec{
			Names: []*ast.Ident{{Name: strcase.ToCamel(b.Name)}},
			Values: []ast.Expr{
				&ast.CompositeLit{
					Type: &ast.Ident{Name: reflect.TypeOf(b).Name()},
					Elts: fields,
				},
			},
		})
	}

	return out
}

func main() {
	blocks, err := downloadInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(`// Package block stores information about blocks in Minecraft.
package block

import (
	"math"
)

// BitsPerBlock indicates how many bits are needed to represent all possible
// block states. This value is used to determine the size of the global palette.
var BitsPerBlock = int(math.Ceil(math.Log2(float64(len(StateID)))))

// ID describes the numeric ID of a block.
type ID uint32

// Block describes information about a type of block.
type Block struct {
	ID          ID
	DisplayName string
	Name        string

	Hardness   float64
	Diggable   bool
	DropIDs    []uint32
	NeedsTools map[uint32]bool

	MinStateID uint32
	MaxStateID uint32

	Transparent      bool
	FilterLightLevel int
	EmitLightLevel   int
}

`)
	format.Node(os.Stdout, token.NewFileSet(), makeBlockDeclaration(blocks))

	fmt.Println()
	fmt.Println()
	fmt.Println("// ByID is an index of minecraft blocks by their ID.")
	fmt.Println("var ByID = map[ID]*Block{")
	for _, b := range blocks {
		fmt.Printf("  %d: &%s,\n", b.ID, strcase.ToCamel(b.Name))
	}
	fmt.Println("}")

	fmt.Println()
	fmt.Println("// StateID maps all possible state IDs to a corresponding block ID.")
	fmt.Println("var StateID = map[uint32]ID{")
	for _, b := range blocks {
		if b.MinStateID == b.MaxStateID {
			fmt.Printf("  %d: %d,\n", b.MinStateID, b.ID)
		} else {
			for i := b.MinStateID; i <= b.MaxStateID; i++ {
				fmt.Printf("  %d: %d,\n", i, b.ID)
			}
		}
	}
	fmt.Println("}")
}