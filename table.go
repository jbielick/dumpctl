package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/heimdalr/dag"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

type Table struct {
	Name     string
	Limit    int
	Where    string
	Rules    []Rule
	Columns  map[string]*Column
	Vertex   *dag.Vertexer
	Block    *hcl.Block
	Database *Database
}

var tableSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "where",
			Required: false,
		},
		{
			Name:     "limit",
			Required: false,
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "rule",
			LabelNames: []string{"name"},
		},
	},
}

func NewTable(db *Database, name string, block *hcl.Block) (table *Table, diags hcl.Diagnostics) {
	table = &Table{
		Name:     name,
		Database: db,
		Block:    block,
		Columns:  make(map[string]*Column),
	}
	tableContent, diags := table.Block.Body.Content(tableSchema)
	if diags.HasErrors() {
		return
	}
	moreDiags := table.SetLimitFromAttributes(tableContent.Attributes)
	if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
		return
	}
	// table.ParseWhere(tableContent.Attributes)
	return
}

func (t *Table) ReadRules() (diags hcl.Diagnostics) {
	tableContent, diags := t.Block.Body.Content(tableSchema)
	if diags.HasErrors() {
		return
	}
	for _, ruleBlock := range tableContent.Blocks {
		_, moreDiags := t.AddRule(ruleBlock.Labels[0], ruleBlock)
		if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
			continue
		}
	}
	return
}

func (t *Table) AddRule(ruleType string, block *hcl.Block) (rule Rule, diags hcl.Diagnostics) {
	ctx := t.EvalContext()
	switch ruleType {
	case "mask":
		rule, diags = NewMaskRule(block, ctx)
	case "redact":
		rule, diags = NewRedactRule(block, ctx)
	default:
		attrRange := block.DefRange
		return nil, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("%s is not a recognized rule type", ruleType),
				Subject:  &attrRange,
			},
		}
	}

	if diags.HasErrors() {
		return nil, diags
	}

	t.Rules = append(t.Rules, rule)

	return
}

func (t *Table) ReadSchema() (diags hcl.Diagnostics) {
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/information_schema",
			t.Database.Config.Options.User,
			t.Database.Config.Options.Password,
			t.Database.Config.Options.Host,
			t.Database.Config.Options.Port,
		),
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer db.Close()
	rows, err := db.Query(`
SELECT COLUMN_NAME, DATA_TYPE, ORDINAL_POSITION, CHARACTER_MAXIMUM_LENGTH
from INFORMATION_SCHEMA.COLUMNS
where TABLE_SCHEMA = ? and TABLE_NAME = ?
order by ORDINAL_POSITION asc`, t.Database.Name, t.Name)
	if err != nil {
		diags.Append(&hcl.Diagnostic{Summary: err.Error()})
	}
	defer rows.Close()

	for rows.Next() {
		var column Column
		if err := rows.Scan(&column.Name, &column.Type, &column.Position, &column.MaxLength); err != nil {
			diags.Append(&hcl.Diagnostic{Summary: err.Error()})
			continue
		}
		column.Table = t
		t.Columns[column.Name] = &column
	}
	if err = rows.Err(); err != nil {
		diags.Append(&hcl.Diagnostic{Summary: err.Error()})
	}
	return
}

func (t *Table) SetLimitFromAttributes(attributes hcl.Attributes) (diags hcl.Diagnostics) {
	attr, ok := attributes["limit"]
	if !ok {
		return nil
	}

	value, moreDiags := attr.Expr.Value(nil)
	if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
		return
	}

	if err := gocty.FromCtyValue(value, &t.Limit); err != nil {
		diags.Append(&hcl.Diagnostic{Summary: err.Error(), Severity: hcl.DiagError})
		return
	}

	return
}

func (t *Table) ParseWhere(attributes hcl.Attributes) (diags hcl.Diagnostics) {
	attr, ok := attributes["where"]
	if !ok {
		return nil
	}
	for _, variable := range attr.Expr.Variables() {
		log.Printf("DEBUG: %+v\n", variable.SimpleSplit().Abs)
		log.Printf("DEBUG: %+v\n", t.Database.Tables[variable.SimpleSplit().Abs.RootName()])
	}
	return
}

func (t *Table) ContextVariables() map[string]cty.Value {
	vars := make(map[string]cty.Value)
	for _, column := range t.Columns {
		vars[column.Name] = cty.StringVal(column.Name)
	}
	return vars
}

func (t *Table) EvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{Variables: t.ContextVariables()}
}
