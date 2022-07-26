package main

import (
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/pingcap/tidb/types"
	driver "github.com/pingcap/tidb/types/parser_driver"
	"github.com/zclconf/go-cty/cty/gocty"
)

type RedactRule struct {
	Columns []string `cty:"columns"`
}

var redactRuleDefaultSpec = hcldec.ObjectSpec{
	"columns": columnSpec,
}

func (r *RedactRule) Apply(row *Row) error {
	for _, columnName := range r.Columns {
		column, ok := row.Table.Columns[columnName]
		if !ok {
			continue
		}

		currentValueExpr := (*row.Values)[column.Position-1]

		if expr, ok := currentValueExpr.(*driver.ValueExpr); ok {
			switch expr.Kind() {
			case types.KindInt64, types.KindUint64, types.KindFloat32, types.KindFloat64:
				expr.Datum.SetValue(0, &expr.Type)
			case types.KindString, types.KindBytes, types.KindMysqlTime:
				expr.Datum.SetValue("", &expr.Type)
			case types.KindMysqlDecimal, types.KindBinaryLiteral,
				types.KindMysqlDuration, types.KindMysqlEnum,
				types.KindMysqlBit, types.KindMysqlSet,
				types.KindInterface, types.KindMinNotNull, types.KindMaxValue,
				types.KindRaw, types.KindMysqlJSON:
				// TODO implement Restore function
				return errors.New("Not implemented")
			case types.KindNull:
				continue
			default:
				return fmt.Errorf("don't know how to redact column %s with type %T", columnName, expr.Kind())
			}
		}
	}
	return nil
}

func NewRedactRule(block *hcl.Block, ctx *hcl.EvalContext) (*RedactRule, hcl.Diagnostics) {
	rule := &RedactRule{}
	decodedSpec, diagnostics := hcldec.Decode(block.Body, redactRuleDefaultSpec, ctx)
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
	return rule, diagnostics
}
