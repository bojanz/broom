// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/adler32"
	"net/http"
	"os"
	"sort"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ghodss/yaml"
	"github.com/iancoleman/strcase"
)

// LoadOperations loads available operations from a specification on disk.
func LoadOperations(filename string) (Operations, error) {
	spec, err := loadSpec(filename)
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}
	if err := spec.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate spec: %w", err)
	}
	// Pre-sort the path map to ensure a consistent ordering of operations.
	paths := make([]string, 0, len(spec.Paths))
	for path := range spec.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	operations := Operations{}
	for _, path := range paths {
		pathItem := spec.Paths[path]
		if pathItem.Get != nil {
			operation := newOperationFromSpec(http.MethodGet, path, pathItem.Parameters, *pathItem.Get)
			operations = append(operations, operation)
		}
		if pathItem.Post != nil {
			operation := newOperationFromSpec(http.MethodPost, path, pathItem.Parameters, *pathItem.Post)
			operations = append(operations, operation)
		}
		if pathItem.Put != nil {
			operation := newOperationFromSpec(http.MethodPut, path, pathItem.Parameters, *pathItem.Put)
			operations = append(operations, operation)
		}
		if pathItem.Patch != nil {
			operation := newOperationFromSpec(http.MethodPatch, path, pathItem.Parameters, *pathItem.Patch)
			operations = append(operations, operation)
		}
		if pathItem.Delete != nil {
			operation := newOperationFromSpec(http.MethodDelete, path, pathItem.Parameters, *pathItem.Delete)
			operations = append(operations, operation)
		}
	}

	return operations, nil
}

// loadSpec loads an OpenAPI 2.0/3.0 specification from disk.
func loadSpec(filename string) (*openapi3.T, error) {
	openapi3.DefineStringFormat("uuid", openapi3.FormatOfStringForUUIDOfRFC4122)
	openapi3.DefineStringFormat("ulid", `^[0-7]{1}[0-9A-HJKMNP-TV-Z]{25}$`)
	openapi3.SchemaFormatValidationDisabled = true

	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	aux := struct {
		OpenAPI string `json:"openapi"`
		Swagger string `json:"swagger"`
	}{}
	// We don't care if unmarshaling fails at this point, we'll assume
	// OpenAPI 3.0 and let openapi3.Loader report the actual problem.
	_ = yaml.Unmarshal(b, &aux)

	var spec *openapi3.T
	if aux.Swagger != "" {
		var spec2 *openapi2.T
		if err := yaml.Unmarshal(b, &spec2); err != nil {
			return nil, fmt.Errorf("v2: %w", err)
		}
		spec, err = openapi2conv.ToV3(spec2)
		if err != nil {
			return nil, fmt.Errorf("v2 to v3: %w", err)
		}
	} else {
		spec, err = openapi3.NewLoader().LoadFromData(b)
		if err != nil {
			return nil, fmt.Errorf("v3: %w", err)
		}
	}

	return spec, nil
}

// newOperationFromSpec creates a new operation from the loaded specification.
func newOperationFromSpec(method string, path string, params openapi3.Parameters, specOp openapi3.Operation) Operation {
	op := Operation{
		ID:          strcase.ToKebab(specOp.OperationID),
		Summary:     specOp.Summary,
		Description: Sanitize(specOp.Description),
		Method:      method,
		Path:        path,
		Deprecated:  specOp.Deprecated,
	}
	// Make it possible to run operations without an ID.
	if op.ID == "" {
		// A hash like c5430c97 is better than nothing, though in the future we
		// could try to generate a more user-friendly machine name from the path.
		hash := adler32.New()
		hash.Write([]byte(path))
		op.ID = hex.EncodeToString(hash.Sum(nil))
	}
	if len(specOp.Tags) > 0 {
		op.Tag = specOp.Tags[0]
	}
	// Parameters can be defined per-path or per-operation..
	op.Parameters = make(Parameters, 0, len(params)+len(specOp.Parameters))
	for _, param := range params {
		op.Parameters = append(op.Parameters, newParameterFromSpec(*param.Value))
	}
	for _, param := range specOp.Parameters {
		op.Parameters = append(op.Parameters, newParameterFromSpec(*param.Value))
	}
	if specOp.RequestBody != nil {
		for format, mediaType := range specOp.RequestBody.Value.Content {
			op.BodyFormat = format
			// Sort the property names to ensure a consistent order.
			names := make([]string, 0, len(mediaType.Schema.Value.Properties))
			for name := range mediaType.Schema.Value.Properties {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
				schema := mediaType.Schema.Value.Properties[name]
				required := false
				for _, requiredName := range mediaType.Schema.Value.Required {
					if requiredName == name {
						required = true
					}
				}

				op.Parameters = append(op.Parameters, Parameter{
					In:          "body",
					Name:        name,
					Description: Sanitize(schema.Value.Description),
					Type:        getSchemaType(*schema.Value),
					Enum:        castEnum(schema.Value.Enum),
					Default:     schema.Value.Default,
					Deprecated:  schema.Value.Deprecated,
					Required:    required,
				})
			}
			break
		}
	}

	return op
}

// newParameterFromSpec creates a new parameter from the loaded specification.
func newParameterFromSpec(specParam openapi3.Parameter) Parameter {
	return Parameter{
		In:          specParam.In,
		Name:        specParam.Name,
		Description: Sanitize(specParam.Description),
		Style:       specParam.Style,
		Type:        getSchemaType(*specParam.Schema.Value),
		Enum:        castEnum(specParam.Schema.Value.Enum),
		Default:     specParam.Schema.Value.Default,
		Deprecated:  specParam.Deprecated,
		Required:    specParam.Required,
	}
}

// getSchemaType retrieves the type of the given schema.
func getSchemaType(schema openapi3.Schema) string {
	schemaType := schema.Type
	// CastString() needs to know the underlying type (array -> []string).
	if schemaType == "array" {
		schemaType = fmt.Sprintf("[]%v", schema.Items.Value.Type)
	}
	return schemaType
}

// castEnum converts enum values from interface{} to string.
func castEnum(enum []interface{}) []string {
	if len(enum) == 0 {
		return nil
	}
	stringEnum := make([]string, 0, len(enum))
	for _, v := range enum {
		stringEnum = append(stringEnum, fmt.Sprintf("%v", v))
	}
	return stringEnum
}
