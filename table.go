package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/heimdalr/dag"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-cty/cty/gocty"
)

var dialect = goqu.Dialect("mysql")

type Where struct {
	Reference *Column
	Value     cty.Value
}

type Table struct {
	Name        string  `hcl:"name,label"`
	SimpleWhere string  `hcl:"where,optional"`
	Limit       int     `hcl:"limit,optional"`
	Order       string  `hcl:"order,optional"`
	SampleRate  float64 `hcl:"sample_rate,optional"`
	Rules       []Rule
	Columns     map[string]*Column
	Body        hcl.Body `hcl:",remain"`
	BodyContent *hcl.BodyContent
	Database    *Database
	Wheres      []map[string]*Where
	OutFile     *os.File
	Dumped      bool
}

var tableSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "where",
		},
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
		Columns:  make(map[string]*Column),
	}

	diags = gohcl.DecodeBody(block.Body, table.EvalContext(true), table)
	tableContent, moreDiags := table.Body.Content(tableSchema)
	diags = append(diags, moreDiags...)
	table.BodyContent = tableContent

	return
}

func (t *Table) ReadDynamicConfig() (diags hcl.Diagnostics) {
	for _, ruleBlock := range t.BodyContent.Blocks.OfType("rule") {
		_, moreDiags := t.AddRule(ruleBlock.Labels[0], ruleBlock)
		if diags = append(diags, moreDiags...); moreDiags.HasErrors() {
			continue
		}
	}
	return t.TrackDependencies(t.BodyContent.Blocks.OfType("where"))
}

func (t *Table) AddRule(ruleType string, block *hcl.Block) (rule Rule, diags hcl.Diagnostics) {
	ctx := t.EvalContext(false)
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
	log.Printf("DEBUG: reading schema for %s.%s\n", t.Database.Name, t.Name)
	rows, err := t.Database.Config.Conn.Query(`
SELECT COLUMN_NAME, DATA_TYPE, ORDINAL_POSITION, CHARACTER_MAXIMUM_LENGTH
from INFORMATION_SCHEMA.COLUMNS
where TABLE_SCHEMA = ? and TABLE_NAME = ?
order by ORDINAL_POSITION asc`, t.Database.Name, t.Name)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{Summary: err.Error(), Severity: hcl.DiagError})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var column Column
		if err := rows.Scan(&column.Name, &column.Type, &column.Position, &column.MaxLength); err != nil {
			diags = diags.Append(&hcl.Diagnostic{Summary: err.Error(), Severity: hcl.DiagError})
			continue
		}
		column.Table = t
		t.Columns[column.Name] = &column
	}

	if err = rows.Err(); err != nil {
		diags = diags.Append(&hcl.Diagnostic{Summary: err.Error(), Severity: hcl.DiagError})
	}
	if len(t.Columns) == 0 {
		diags = diags.Append(&hcl.Diagnostic{Summary: fmt.Sprintf("Could not read schema of %s. Is it misspelled?", t.Name), Severity: hcl.DiagError})
	}
	return
}

func (t *Table) CustomBodySchema() (schema *hcl.BodySchema) {
	schema = &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{},
	}
	for colName, _ := range t.Columns {
		schema.Attributes = append(schema.Attributes, hcl.AttributeSchema{
			Name:     colName,
			Required: false,
		})
	}
	return
}

func (t *Table) TrackDependencies(wheres []*hcl.Block) (diags hcl.Diagnostics) {
	log.Printf("DEBUG: building dependency graph for %+v\n", t.Name)

	for _, block := range wheres {
		whereContent, moreDiags := block.Body.Content(t.CustomBodySchema())
		diags = append(diags, moreDiags...)
		whereGroup := make(map[string]*Where)
		for colName, attr := range whereContent.Attributes {
			variables := attr.Expr.Variables()
			if len(variables) > 1 {
				diags = diags.Append(&hcl.Diagnostic{
					Summary:  "Cannot make more than one table reference in where condition",
					Severity: hcl.DiagError,
					Subject:  attr.Expr.Range().Ptr(),
				})
				continue
			}
			if len(variables) == 0 {
				value, moreDiags := attr.Expr.Value(nil)
				diags = append(diags, moreDiags...)
				if moreDiags.HasErrors() {
					continue
				}
				whereGroup[colName] = &Where{Value: value}
			} else {
				reference, moreDiags := variables[0].TraverseAbs(t.Database.EvalContext())
				diags = append(diags, moreDiags...)
				if moreDiags.HasErrors() {
					continue
				}
				parts := strings.Split(reference.AsString(), ".")
				tableRefName, columnRefName := parts[0], parts[1]

				if err := t.Database.DAG.AddEdge(tableRefName, t.Name); err != nil {
					var severity hcl.DiagnosticSeverity
					switch err.(type) {
					case dag.EdgeLoopError:
						severity = hcl.DiagError
						err = fmt.Errorf("table `%s` dependency on `%s` would create a circular dependency", t.Name, tableRefName)
					case dag.EdgeDuplicateError:
						continue
					default:
						severity = hcl.DiagError
					}
					diags = diags.Append(&hcl.Diagnostic{
						Summary:  err.Error(),
						Severity: severity,
						Subject:  attr.Expr.Range().Ptr(),
					})
				}
				whereGroup[colName] = &Where{
					Reference: t.Database.Tables[tableRefName].Columns[columnRefName],
				}
			}
		}
		t.Wheres = append(t.Wheres, whereGroup)
	}
	return
}

func (t *Table) Select(selectCol string) string {
	expressions := []goqu.Expression{}
	if len(t.SimpleWhere) != 0 {
		expressions = append(expressions, goqu.L(t.SimpleWhere))
	}
	if t.SampleRate > 0 {
		expressions = append(expressions, goqu.L(fmt.Sprintf("(crc32(id) %% %d = 0)", int(1/t.SampleRate))))
	}

	orExpressions := []goqu.Expression{}
	for _, whereGroup := range t.Wheres {
		conditions := make(goqu.Ex)

		for colName, where := range whereGroup {
			if where.Reference != nil {
				conditions[colName] = goqu.Op{
					"in": goqu.L(fmt.Sprintf("(select * from (%s) _tmp_%s)", where.Reference.Table.Select(where.Reference.Name), where.Reference.Table.Name)),
				}
			} else if where.Value.Type() == cty.String {
				var s string
				err := gocty.FromCtyValue(where.Value, &s)
				if err != nil {
					log.Fatalf(err.Error())
				}
				conditions[colName] = s
			} else if where.Value.Type() == cty.Number {
				var i int64
				err := gocty.FromCtyValue(where.Value, &i)
				if err != nil {
					log.Fatalf(err.Error())
				}
				conditions[colName] = i
			} else if where.Value.Type().IsTupleType() {
				log.Fatalf("don't know how to decode tuple yet")
			} else if where.Value.Type() == cty.NilType {
				conditions[colName] = nil
			}
		}
		orExpressions = append(orExpressions, conditions)
	}

	sql, _, _ := dialect.From(t.Name).Select(selectCol).Where(
		append(expressions, goqu.Or(orExpressions...))...,
	).ToSQL()

	if len(t.Order) != 0 {
		sql = fmt.Sprintf("%s order by %s", sql, t.Order)
	}
	if t.Limit > 0 {
		sql = fmt.Sprintf("%s limit %d", sql, t.Limit)
	}

	return sql
}

func (t *Table) Where() string {
	fullSelect := t.Select("*")
	emptySelect, _, _ := dialect.From(t.Name).ToSQL()

	return strings.Replace(
		strings.Replace(
			fullSelect,
			fmt.Sprintf("%s", emptySelect),
			"", 1,
		),
		" WHERE ",
		"", 1,
	)
}

func (t *Table) ContextVariables(withTablePrefix bool) map[string]cty.Value {
	vars := make(map[string]cty.Value)
	for _, column := range t.Columns {
		if withTablePrefix {
			vars[column.Name] = cty.StringVal(fmt.Sprintf("%s.%s", column.Table.Name, column.Name))
		} else {
			vars[column.Name] = cty.StringVal(column.Name)
		}
	}
	return vars
}

func (t *Table) EvalContext(withTablePrefix bool) *hcl.EvalContext {
	now := function.New(&function.Spec{
		Params: []function.Parameter{},
		Type:   function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			ts := t.Database.Config.Started.Format(time.RFC3339)
			return cty.StringVal(ts), nil
		}})
	return &hcl.EvalContext{
		Variables: t.ContextVariables(withTablePrefix),
		Functions: map[string]function.Function{
			"timeadd": stdlib.TimeAddFunc,
			"now":     now,
		},
	}
}

func (t *Table) String() string {
	return fmt.Sprintf("`%s`", t.Name)
}
