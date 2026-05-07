package tools

import (
	"fmt"

	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// resolveRef constructs an *adt.ObjectRef from MCP tool arguments.
// Accepts both human-readable IDs ("class", "cds_view") and short codes ("CLAS", "DDLS")
// for backward compatibility.
func resolveRef(objectType, name string, parent, include, method string) (*adt.ObjectRef, error) {
	if objectType == "" {
		return nil, fmt.Errorf("object_type is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	var opts []adt.RefOption
	if parent != "" {
		opts = append(opts, adt.WithParent(parent))
	}
	if include != "" {
		opts = append(opts, adt.WithInclude(include))
	}
	if method != "" {
		opts = append(opts, adt.WithMethod(method))
	}

	return adt.NewObjectRef(objectType, name, opts...)
}

// resolveRefSimple constructs an *adt.ObjectRef with just type and name.
func resolveRefSimple(objectType, name string) (*adt.ObjectRef, error) {
	return resolveRef(objectType, name, "", "", "")
}
