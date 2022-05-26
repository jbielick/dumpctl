package main

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/heimdalr/dag"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type Database struct {
	Name   string
	Tables map[string]*Table
	DAG    *dag.DAG
	Block  *hcl.Block
	Config *Config
}

var databaseSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "table",
			LabelNames: []string{"name"},
		},
	},
}

func NewDatabase(name string, block *hcl.Block, config *Config) (database *Database, diags hcl.Diagnostics) {
	database = &Database{
		Name:   name,
		Block:  block,
		Tables: make(map[string]*Table),
		DAG:    dag.NewDAG(),
		Config: config,
	}

	content, diags := database.Block.Body.Content(databaseSchema)
	if diags.HasErrors() {
		return
	}

	for _, tableBlock := range content.Blocks {
		_, moreDiags := database.AddTable(tableBlock.Labels[0], tableBlock)
		if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
			continue
		}
	}
	return
}

func (d *Database) AddTable(name string, block *hcl.Block) (table *Table, diags hcl.Diagnostics) {
	table, diags = NewTable(d, name, block)
	if diags.HasErrors() {
		return
	}
	d.Tables[name] = table
	d.DAG.AddVertex(table)
	return
}

func (d *Database) ReadSchema() (diags hcl.Diagnostics) {
	for _, table := range d.Tables {
		moreDiags := table.ReadSchema()
		if diags = append(diags, moreDiags...); diags.HasErrors() {
			continue
		}
	}
	return diags
}

func (d *Database) ReadRules() (diags hcl.Diagnostics) {
	for _, table := range d.Tables {
		moreDiags := table.ReadRules()
		if diags = append(diags, moreDiags...); diags.HasErrors() {
			continue
		}
	}
	return diags
}

func (d *Database) ContextVariables() map[string]cty.Value {
	vars := make(map[string]cty.Value)
	for _, table := range d.Tables {
		vars[table.Name] = cty.MapVal(table.ContextVariables())
	}
	return vars
}

func (d *Database) EvalContext() *hcl.EvalContext {
	selectFn := function.New(&function.Spec{
		Params: []function.Parameter{
			{Name: "column", Type: cty.String},
			{Name: "table", Type: cty.DynamicPseudoType},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {

			return cty.StringVal(
				fmt.Sprintf(
					"select %s from %s where %s",
					args[0].AsString(),
					args[1].AsValueMap()["__name"].AsString(),
					"1",
				),
			), nil
		},
	})
	return &hcl.EvalContext{
		Variables: d.ContextVariables(),
		Functions: map[string]function.Function{
			"select": selectFn,
		},
	}
}
