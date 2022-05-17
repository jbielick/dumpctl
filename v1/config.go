package v1

import (
	"gopkg.in/yaml.v2"
)

type Database struct {
	Name           string `yaml:"name"`
	IncludedTables tables `yaml:"include"`
	// ExcludedTables tables `yaml:"exclude"`
}

type tables []table

func (ts *tables) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tableMap map[string]table
	if err := unmarshal(&tableMap); err != nil {
		return err
	}
	for name, table := range tableMap {
		table.Name = name
		*ts = append(*ts, table)
	}
	return nil
}

type table struct {
	Name    string
	Columns []column
}

func (t *table) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var columnMap map[string]column
	var columnShortcutConfigMap map[string]string
	if err := unmarshal(&columnMap); err != nil {
		if _, ok := err.(*yaml.TypeError); !ok {
			return err
		}
		if err := unmarshal(&columnShortcutConfigMap); err != nil {
			return err
		}
	}
	var columns []column
	if len(columnMap) > 0 {
		for name, column := range columnMap {
			column.Name = name
			columns = append(columns, column)
		}
	} else {
		for name, ruleName := range columnShortcutConfigMap {
			columns = append(columns, column{Name: name, Rules: []rule{
				{Name: ruleName},
			}})
		}
	}

	t.Columns = columns

	return nil
}

type column struct {
	Name  string
	Rules []rule
}

type rule struct {
	Name string
}

type config struct {
	Databases []Database `yaml:"databases"`
}

func NewConfig() *config {
	return &config{}
}
