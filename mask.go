package main

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	driver "github.com/pingcap/tidb/types/parser_driver"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

type MaskRule struct {
	Columns       []string `cty:"columns"`
	Surrogate     string   `cty:"surrogate"`
	PatternString string   `cty:"pattern"`
	Pattern       *regexp.Regexp
}

var maskRuleDefaultSpec = hcldec.ObjectSpec{
	"columns": columnSpec,
	"surrogate": &hcldec.DefaultSpec{
		Primary: &hcldec.AttrSpec{
			Name: "surrogate",
			Type: cty.String,
		},
		Default: &hcldec.LiteralSpec{Value: cty.StringVal("*")},
	},
	"pattern": &hcldec.DefaultSpec{
		Primary: &hcldec.AttrSpec{
			Name: "pattern",
			Type: cty.String,
		},
		Default: &hcldec.LiteralSpec{Value: cty.StringVal(`[^\s]`)},
	},
}

func (r *MaskRule) Apply(row *Row) error {
	for _, columnName := range r.Columns {
		column, ok := row.Table.Columns[columnName]
		if !ok {
			continue
		}

		currentValueExpr := (*row.Values)[column.Position-1]

		if expr, ok := currentValueExpr.(*driver.ValueExpr); ok {
			s, _ := expr.Datum.ToString()
			expr.Datum.SetValue(r.Pattern.ReplaceAllString(s, r.Surrogate), &expr.Type)
		}
	}
	return nil
}

func NewMaskRule(block *hcl.Block, ctx *hcl.EvalContext) (*MaskRule, hcl.Diagnostics) {
	rule := &MaskRule{}
	decodedSpec, diagnostics := hcldec.Decode(block.Body, maskRuleDefaultSpec, ctx)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	err := gocty.FromCtyValue(decodedSpec, &rule)
	if err != nil {
		attrRange := block.Body.MissingItemRange()
		return nil, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("error while configuring %s rule: %v", "@TODO", err.Error()),
				Subject:  &attrRange,
			},
		}
	}
	rule.Pattern = regexp.MustCompile(rule.PatternString)
	return rule, diagnostics
}
