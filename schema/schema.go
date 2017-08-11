package schema

import (
	"fmt"
	"github.com/zimnx/YamlSchemaToGoStruct/item"
	"github.com/zimnx/YamlSchemaToGoStruct/set"
	"github.com/zimnx/YamlSchemaToGoStruct/util"
)

// Schema is a for a gohan schema
type Schema struct {
	parent  string
	extends []string
	schema  *item.Property
}

func prepareSchema(data map[interface{}]interface{}) map[interface{}]interface{} {
	if len(data) != 0 {
		return data
	}
	return map[interface{}]interface{}{
		"type": "object",
	}
}

// Name is a function that allows schema to be used as a set element
func (schema *Schema) Name() string {
	return schema.schema.Name()
}

func (schema *Schema) bases() []string {
	return schema.extends
}

func (schema *Schema) getName(data map[interface{}]interface{}) error {
	id, ok := data["id"].(string)
	if !ok {
		return fmt.Errorf("schema does not have an id")
	}
	schema.schema = item.CreateProperty(id)
	return nil
}

func (schema *Schema) getParent(data map[interface{}]interface{}) {
	schema.parent, _ = data["parent"].(string)
}

func (schema *Schema) getBaseSchemas(data map[interface{}]interface{}) error {
	extends, ok := data["extends"].([]interface{})
	if !ok {
		return nil
	}
	bases := make([]string, len(extends))
	for i, base := range extends {
		bases[i], ok = base.(string)
		if !ok {
			return fmt.Errorf("one of the base schemas is not a string")
		}
	}
	schema.extends = bases
	return nil
}

func (schema *Schema) addParent() error {
	if schema.parent == "" {
		return nil
	}
	data := map[interface{}]interface{}{
		"type": "string",
	}
	property := item.CreateProperty(util.AddName(schema.parent, "id"))
	property.Parse("", 0, true, data)
	set := set.New()
	set.Insert(property)
	return schema.schema.AddProperties(set, true)
}

func (schema *Schema) parse(data map[interface{}]interface{}) error {
	if err := schema.getName(data); err != nil {
		return err
	}
	schema.getParent(data)
	if err := schema.getBaseSchemas(data); err != nil {
		return fmt.Errorf(
			"invalid schema %s: %v",
			schema.schema.Name(),
			err,
		)
	}
	next, ok := data["schema"].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf(
			"invalid schema %s: schema does not have a \"schema\"",
			schema.Name(),
		)
	}
	next = prepareSchema(next)
	if err := schema.schema.Parse("", 0, true, next); err != nil {
		return fmt.Errorf(
			"invalid schema %s: %v",
			schema.Name(),
			err,
		)
	}
	if !schema.schema.IsObject() {
		return fmt.Errorf(
			"invalid schema %s: schema should be an object",
			schema.Name(),
		)
	}
	err := schema.addParent()
	if err != nil {
		return fmt.Errorf("invalid schema %s: %v",
			schema.Name(),
			err,
		)
	}
	return nil
}

func (schema *Schema) collectObjects(limit, offset int) (set.Set, error) {
	result, err := schema.schema.CollectObjects(limit, offset)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid schema %s: %v",
			schema.Name(),
			err,
		)
	}
	return result, nil
}

func (schema *Schema) collectProperties(limit, offset int) (set.Set, error) {
	result, err := schema.schema.CollectProperties(limit, offset)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid schema %s: %v",
			schema.Name(),
			err,
		)
	}
	return result, nil
}

func (schema *Schema) join(edges []*node) error {
	properties := set.New()
	for _, node := range edges {
		// Impossible to have error here
		newProperties, _ := node.value.collectProperties(2, 1)
		if err := properties.SafeInsertAll(newProperties); err != nil {
			return fmt.Errorf(
				"multiple properties with the same name in bases of schema %s",
				schema.Name(),
			)
		}
	}
	err := schema.schema.AddProperties(properties, false)
	if err != nil {
		return fmt.Errorf(
			"schema %s should be an object",
			schema.Name(),
		)
	}
	return nil
}

func parseAll(schemas []map[interface{}]interface{}) (set.Set, error) {
	set := set.New()
	for _, object := range schemas {
		newSchema := &Schema{}
		if err := newSchema.parse(object); err != nil {
			return nil, err
		}
		if err := set.SafeInsert(newSchema); err != nil {
			return nil, fmt.Errorf(
				"multiple schemas with the same name: %s",
				newSchema.Name(),
			)
		}
	}
	return set, nil
}
