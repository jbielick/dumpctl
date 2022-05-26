package main

import (
	"github.com/hashicorp/hcl/v2/hcldec"

	"github.com/zclconf/go-cty/cty"
)

var columnSpec = &hcldec.AttrSpec{
	Name:     "columns",
	Type:     cty.List(cty.String),
	Required: true,
}

type Rule interface {
	Apply(*Row) error
}

// @TODO
// Replace (Faker?)
// Tokenize
// Bucketing
// Date Shifting
// Time extraction
