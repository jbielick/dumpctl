package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/heimdalr/dag"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
	_ "github.com/pingcap/tidb/types/parser_driver"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

type DumpSequencer struct {
	Config *Config
	Dumper *Dumper
}

func NewDumpSequencer(config *Config) (*DumpSequencer, error) {
	dumper, err := NewDumper(config.Options)
	return &DumpSequencer{Config: config, Dumper: dumper}, err
}

func (s *DumpSequencer) DumpDatabase(database *Database) error {
	dumper, err := NewDumper(s.Config.Options)
	if err != nil {
		return err
	}
	for _, table := range database.Tables {
		outFile, err := ioutil.TempFile("", table.Name)
		if err != nil {
			log.Fatal(err)
		}
		table.OutFile = outFile
		defer os.Remove(table.OutFile.Name())
	}
	var diags hcl.Diagnostics
	visitor := &DBDAGVisitor{
		Database:    database,
		Dumper:      dumper,
		Parser:      parser.New(),
		Diagnostics: diags,
		Wg:          &sync.WaitGroup{},
	}
	dag.BFSWalk(database.DAG, visitor)
	var sb strings.Builder
	wr := hcl.NewDiagnosticTextWriter(&sb, nil, 78, true)
	wr.WriteDiagnostics(visitor.Diagnostics)
	if diags.HasErrors() {
		err = fmt.Errorf("Could not read config")
	}
	log.Println(sb.String())
	return nil
}

func (s *DumpSequencer) Dump() error {
	for _, database := range s.Config.Databases {
		if err := s.DumpDatabase(database); err != nil {
			return err
		}
	}
	return nil
}

type RuleVisitor struct {
	Table *Table
}

func (v *RuleVisitor) Enter(in ast.Node) (ast.Node, bool) {
	if stmt, ok := in.(*ast.InsertStmt); ok {
		valuesExpr := stmt.Lists[0]

		row := &Row{
			Table:  v.Table,
			Values: &valuesExpr,
		}
		for _, rule := range v.Table.Rules {
			err := rule.Apply(row)
			if err != nil {
				// @TODO continue and collect errors?
				log.Fatal(err.Error())
			}
		}
	}
	return in, true
}

func (v *RuleVisitor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

type DBDAGVisitor struct {
	Database    *Database
	Dumper      *Dumper
	Diagnostics hcl.Diagnostics
	Parser      *parser.Parser
	Wg          *sync.WaitGroup
}

func (v *DBDAGVisitor) Visit(wrapper dag.Vertexer) {
	id, value := wrapper.Vertex()
	table, ok := value.(*Table)
	if !ok {
		return
	}
	allDependenciesMet := true
	ancestors, err := v.Database.DAG.GetAncestors(id)
	if err != nil {
		log.Fatalf(err.Error())
	}
	for _, ancestor := range ancestors {
		dependency, ok := ancestor.(*Table)
		if !ok {
			return
		}
		allDependenciesMet = allDependenciesMet && dependency.Dumped
	}
	if !allDependenciesMet || table.Dumped {
		return
	}

	v.Dumper.Reset()
	v.Dumper.AddTables(v.Database.Name, table.Name)
	v.Dumper.SetExtraOptions(v.Database.Config.Options.ExtraArgs)

	var s string
	if table.WhereExpr != nil {
		diags := gohcl.DecodeExpression(
			table.WhereExpr,
			v.Database.EvalContext(),
			&s,
		)
		v.Diagnostics = append(v.Diagnostics, diags...)
		if v.Diagnostics.HasErrors() {
			return
		}
		v.Dumper.SetWhere(s)
		log.Printf("where: %+v\n", s)

	}
	v.Dumper.SetOrder(table.Order)
	v.Dumper.SetLimit(table.Limit)
	table.OutFile.WriteString(fmt.Sprintf("-- %s\n", table.Name))

	visitor := &RuleVisitor{Table: table}

	r, w := io.Pipe()

	go func() {
		defer w.Close()
		v.Dumper.Dump(w)
	}()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 5 && line[0:6] == "INSERT" {
			stmtNode, err := v.Parser.ParseOneStmt(line, "", "")
			if err != nil {
				log.Fatal(err.Error())
			}

			stmtNode.Accept(visitor)
			stmtNode.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, table.OutFile))
			table.OutFile.Write([]byte(";\n"))
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err.Error())
	}
	table.OutFile.Seek(0, io.SeekStart)
	io.Copy(os.Stdout, table.OutFile)
	table.Dumped = true
	return
}

func (v *DBDAGVisitor) EvalContext(table *Table) *hcl.EvalContext {
	// inFn := function.New(&function.Spec{
	// 	Params: []function.Parameter{
	// 		{Name: "column", Type: cty.String},
	// 		{Name: "list", Type: cty.String},
	// 	},
	// 	Type: function.StaticReturnType(cty.List(cty.String)),
	// 	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
	// 		log.Printf("args: %+v\n", args)

	// 		return cty.StringVal(
	// 			fmt.Sprintf(
	// 				"select %s from %s where %s",
	// 				args[0].AsString(),
	// 				args[1].AsValueMap()["__name"].AsString(),
	// 				"1",
	// 			),
	// 		), nil
	// 	},
	// })
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"partners": cty.ListVal([]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"id": cty.StringVal("1"),
				}),
				cty.MapVal(map[string]cty.Value{
					"id": cty.StringVal("2"),
				}),
				cty.MapVal(map[string]cty.Value{
					"id": cty.StringVal("3"),
				}),
			}),
		},
		Functions: map[string]function.Function{
			"join": stdlib.JoinFunc,
		},
	}

}
