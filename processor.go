package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
	_ "github.com/pingcap/tidb/types/parser_driver"
)

type Processor struct {
	Config *Config
	Wg     *sync.WaitGroup
}

func NewProcessor(config *Config) *Processor {
	var wg sync.WaitGroup
	return &Processor{Config: config, Wg: &wg}
}

func (p *Processor) DumpDatabase(s chan error, ctx context.Context, database *Database) {
	defer p.Wg.Done()
	parser := parser.New()

	dumper, err := NewDumper(p.Config.Options)
	if err != nil {
		s <- err
	}

	for _, table := range database.Tables {
		dumper.Reset()
		dumper.AddTables(database.Name, table.Name)
		dumper.SetExtraOptions(p.Config.Options.ExtraArgs)
		dumper.SetWhere(table.Where)
		dumper.SetLimit(table.Limit)

		visitor := &RuleVisitor{Table: &table}

		r, w := io.Pipe()

		go func() {
			defer w.Close()
			dumper.Dump(w)
		}()

		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				line := scanner.Text()

				if len(line) > 5 && line[0:6] == "INSERT" {
					stmtNode, err := parser.ParseOneStmt(line, "", "")
					if err != nil {
						s <- err
					}

					// fmt.Println("<-", line)
					stmtNode.Accept(visitor)
					var sb strings.Builder
					stmtNode.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
					sb.Write([]byte(";"))
					line = sb.String()
				}
				fmt.Println(line)
			}
		}
		if err := scanner.Err(); err != nil {
			s <- err
		}
	}
}

func (p *Processor) Run() error {
	s := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1*time.Hour))
	defer cancel()

	for _, database := range p.Config.Databases {
		p.Wg.Add(1)
		go p.DumpDatabase(s, ctx, &database)
	}

	go func() {
		p.Wg.Wait()
		s <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-s:
		return err
	}
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
