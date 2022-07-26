package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/heimdalr/dag"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
)

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

type Rewriter struct {
	Database    *Database
	Dumper      *Dumper
	Diagnostics hcl.Diagnostics
	Parser      *parser.Parser
	Wg          *sync.WaitGroup
}

func (r *Rewriter) PrintStatus() {
}

func (r *Rewriter) Visit(wrapper dag.Vertexer) {
	id, value := wrapper.Vertex()
	table, ok := value.(*Table)
	if !ok {
		return
	}
	allDependenciesMet := true
	ancestors, err := r.Database.DAG.GetAncestors(id)
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

	r.Dumper.Reset()
	r.Dumper.AddTables(r.Database.Name, table.Name)
	r.Dumper.SetExtraOptions(r.Database.Config.Options.ExtraArgs)
	r.Dumper.SetWhere(table.Where())

	visitor := &RuleVisitor{Table: table}

	readPipe, writePipe := io.Pipe()

	go func() {
		defer writePipe.Close()
		r.Dumper.Dump(writePipe)
	}()

	scanner := bufio.NewScanner(readPipe)
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 5 && line[0:6] == "INSERT" {
			stmtNode, err := r.Parser.ParseOneStmt(line, "", "")
			if err != nil {
				log.Fatal(err.Error())
			}

			stmtNode.Accept(visitor)
			err = stmtNode.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, table.OutFile))
			if err != nil {
				log.Fatalf(err.Error())
			}
			table.OutFile.Write([]byte(";\n"))
		} else {
			table.OutFile.WriteString(fmt.Sprintf("%s\n", line))
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
