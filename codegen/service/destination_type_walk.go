package service

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	destinationVisit struct {
		attribute *expr.AttributeExpr
		path      string
		wrapped   bool
	}

	destinationTypeCollector struct {
		types map[string]map[string]expr.DataType
		seen  map[destinationVisit]bool
	}
)

// walk inventories exactly the named declarations emitted by collectTypes and
// collectUnionTypes. A named union wrapper owns its underlying union; a bare
// union nested in a relocated definition is a declaration in that destination.
func (c *destinationTypeCollector) walk(att *expr.AttributeExpr, location *codegen.Location, wrapped bool) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	if explicit := codegen.UserTypeLocation(att.Type); explicit != nil {
		location = explicit
	}
	path := ""
	if location != nil {
		path = location.RelImportPath
	}
	visit := destinationVisit{attribute: att, path: path, wrapped: wrapped}
	if c.seen[visit] {
		return
	}
	c.seen[visit] = true
	register := func(typ expr.DataType) {
		if path == "" {
			return
		}
		if c.types[path] == nil {
			c.types[path] = make(map[string]expr.DataType)
		}
		c.types[path][typ.Hash()] = typ
	}
	switch typ := att.Type.(type) {
	case expr.UserType:
		register(typ)
		c.walk(typ.Attribute(), location, expr.AsUnion(typ) != nil)
	case *expr.Union:
		if !wrapped {
			register(typ)
		}
		for _, branch := range typ.Values {
			c.walk(branch.Attribute, location, false)
		}
	case *expr.Object:
		for _, field := range *typ {
			c.walk(field.Attribute, location, false)
		}
		c.walk(att.AdditionalAttributes, location, false)
	case *expr.Array:
		c.walk(typ.ElemType, location, false)
	case *expr.Map:
		c.walk(typ.KeyType, location, false)
		c.walk(typ.ElemType, location, false)
	}
}
