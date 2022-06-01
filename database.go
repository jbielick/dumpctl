package main

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/heimdalr/dag"
	"github.com/zclconf/go-cty/cty"
)

type Database struct {
	Name        string
	Tables      map[string]*Table
	DAG         *dag.DAG
	Block       *hcl.Block
	Config      *Config
	Destination string   `hcl:"destination_database,optional"`
	Remain      hcl.Body `hcl:",remain"`
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

	// allows extra content to be parsed later in Body.PartialContent
	moreDiags := gohcl.DecodeBody(block.Body, nil, database)
	diags = append(diags, moreDiags...)
	if moreDiags.HasErrors() {
		return
	}

	if len(database.Destination) == 0 {
		database.Destination = name
	}

	// partial because some of the attributes may be consumed with DecodeBody
	content, _, diags := database.Block.Body.PartialContent(databaseSchema)
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
	if _, ok := d.Tables[name]; ok {
		return nil, diags.Append(&hcl.Diagnostic{
			Summary:  fmt.Sprintf("cannot add duplicate table '%s'", name),
			Subject:  &block.LabelRanges[0],
			Severity: hcl.DiagError,
		})
	}
	table, diags = NewTable(d, name, block)
	if diags.HasErrors() {
		return
	}
	d.Tables[name] = table
	d.DAG.AddVertexByID(table.Name, table)
	return
}

func (d *Database) ReadSchema() (diags hcl.Diagnostics) {
	for _, table := range d.Tables {
		moreDiags := table.ReadSchema()
		if diags = append(diags, moreDiags...); diags.HasErrors() {
			return
		}
	}
	return
}

func (d *Database) ReadDynamicConfig() (diags hcl.Diagnostics) {
	for _, table := range d.Tables {
		moreDiags := table.ReadDynamicConfig()
		if diags = append(diags, moreDiags...); diags.HasErrors() {
			continue
		}
	}
	return
}

func (d *Database) ContextVariables() map[string]cty.Value {
	vars := make(map[string]cty.Value)
	for _, table := range d.Tables {
		vars[table.Name] = cty.MapVal(table.ContextVariables(true))
	}
	return vars
}

func (d *Database) EvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: d.ContextVariables(),
	}
}
