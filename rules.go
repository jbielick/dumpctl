package main

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"

	"github.com/zclconf/go-cty/cty"
)

var columnSpec = &hcldec.AttrSpec{
	Name:     "columns",
	Type:     cty.List(cty.String),
	Required: true,
}

type ConfiguredRule interface {
	Apply(*Row) error
}

type UnconfiguredRule struct {
	Type     string   `hcl:"type,label"`
	SpecBody hcl.Body `hcl:",remain"`
}

func NewConfiguredRule(table *Table, unconfiguredRule UnconfiguredRule, ctx *hcl.EvalContext) (ConfiguredRule, hcl.Diagnostics) {
	switch unconfiguredRule.Type {
	case "mask":
		return NewMaskRule(unconfiguredRule, ctx)
	case "redact":
		return NewRedactRule(unconfiguredRule, ctx)
	default:
		attrRange := unconfiguredRule.SpecBody.MissingItemRange()
		return nil, hcl.Diagnostics{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("%s is not a recognized rule type", unconfiguredRule.Type),
				Subject:  &attrRange,
			},
		}
	}
}

// @TODO
// Redact
// Replace (Faker?)
// Tokenize
// Bucketing
// Date Shifting
// Time extraction
