package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/pingcap/parser/ast"
	"github.com/zclconf/go-cty/cty"
)

type Config struct {
	Databases []Database `hcl:"database,block"`
	Extra     hcl.Body   `hcl:",remain"`
	Options   *Options
}

type Database struct {
	Name      string  `hcl:"name,label"`
	AllTables *bool   `hcl:"all_tables"`
	Tables    []Table `hcl:"table,block"`
}

type Table struct {
	Name              string             `hcl:"name,label"`
	Extra             hcl.Body           `hcl:",remain"`
	Where             []string           `hcl:"where,optional"`
	UnconfiguredRules []UnconfiguredRule `hcl:"rule,block"`
	Rules             []ConfiguredRule
	Columns           map[string]Column
}

type Column struct {
	Name      string
	Position  int64
	Type      string
	MaxLength sql.NullInt64
}

type Row struct {
	Table  *Table
	Values *[]ast.ExprNode
}

func NewConfig(opts *Options) (config *Config, err error) {
	parser := hclparse.NewParser()
	config = &Config{Options: opts}
	f, diags := parser.ParseHCLFile(opts.ConfigFile)
	moreDiags := gohcl.DecodeBody(f.Body, nil, config)
	diags = append(diags, moreDiags...)
	moreDiags = config.ConfigureRules()
	diags = append(diags, moreDiags...)
	if diags.HasErrors() {
		var sb strings.Builder
		wr := hcl.NewDiagnosticTextWriter(&sb, parser.Files(), 78, true)
		wr.WriteDiagnostics(diags)
		err = fmt.Errorf("%s", sb.String())
	}
	return
}

func (c *Config) GetTableSchema(
	database *Database,
	table *Table,
) (columns map[string]Column, diagnostics hcl.Diagnostics) {
	columns = make(map[string]Column)
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/information_schema",
			c.Options.User,
			c.Options.Password,
			c.Options.Host,
			c.Options.Port,
		),
	)
	defer db.Close()
	if err != nil {
		log.Fatal(err.Error())
	}
	rows, err := db.Query(`
SELECT COLUMN_NAME, DATA_TYPE, ORDINAL_POSITION, CHARACTER_MAXIMUM_LENGTH
from INFORMATION_SCHEMA.COLUMNS
where TABLE_SCHEMA = ? and TABLE_NAME = ?
order by ORDINAL_POSITION asc`, database.Name, table.Name)
	if err != nil {
		diagnostics.Append(&hcl.Diagnostic{Summary: err.Error()})
	}
	defer rows.Close()

	for rows.Next() {
		var column Column
		if err := rows.Scan(&column.Name, &column.Type, &column.Position, &column.MaxLength); err != nil {
			diagnostics.Append(&hcl.Diagnostic{Summary: err.Error()})
		}
		columns[column.Name] = column
	}
	if err = rows.Err(); err != nil {
		diagnostics.Append(&hcl.Diagnostic{Summary: err.Error()})
	}
	return
}

func (c *Config) ConfigureRules() (diagnostics hcl.Diagnostics) {
	for _, database := range c.Databases {
		for tableIndex, table := range database.Tables {
			columns, moreDiags := c.GetTableSchema(&database, &table)
			diagnostics = append(diagnostics, moreDiags...)
			if moreDiags.HasErrors() {
				continue
			}
			database.Tables[tableIndex].Columns = columns
			tableCtx := make(map[string]cty.Value)
			for _, column := range columns {
				tableCtx[column.Name] = cty.StringVal(column.Name)
			}
			for _, rule := range table.UnconfiguredRules {
				ctx := &hcl.EvalContext{
					Variables: map[string]cty.Value{
						"table": cty.MapVal(tableCtx),
					},
				}
				configuredRule, moreDiags := NewConfiguredRule(&database.Tables[tableIndex], rule, ctx)
				diagnostics = append(diagnostics, moreDiags...)
				if moreDiags.HasErrors() {
					continue
				}

				database.Tables[tableIndex].Rules = append(database.Tables[tableIndex].Rules, configuredRule)
			}
		}
	}
	return
}
