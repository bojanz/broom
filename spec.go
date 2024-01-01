// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash/adler32"
	"net/http"
	"os"
	"slices"

	"github.com/iancoleman/strcase"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// LoadOperations loads available operations from a specification on disk.
func LoadOperations(filename string) (Operations, error) {
	spec, err := LoadSpec(filename)
	if err != nil {
		return Operations{}, fmt.Errorf("load spec: %w", err)
	}
	if spec.Paths == nil {
		return Operations{}, nil
	}

	ops := Operations{}
	for pair := orderedmap.First(spec.Paths.PathItems); pair != nil; pair = pair.Next() {
		path := pair.Key()
		pathItem := pair.Value()
		if pathItem.Get != nil {
			ops = append(ops, newOperationFromSpec(http.MethodGet, path, pathItem.Parameters, *pathItem.Get))
		}
		if pathItem.Post != nil {
			ops = append(ops, newOperationFromSpec(http.MethodPost, path, pathItem.Parameters, *pathItem.Post))
		}
		if pathItem.Put != nil {
			ops = append(ops, newOperationFromSpec(http.MethodPut, path, pathItem.Parameters, *pathItem.Put))
		}
		if pathItem.Patch != nil {
			ops = append(ops, newOperationFromSpec(http.MethodPatch, path, pathItem.Parameters, *pathItem.Patch))
		}
		if pathItem.Delete != nil {
			ops = append(ops, newOperationFromSpec(http.MethodDelete, path, pathItem.Parameters, *pathItem.Delete))
		}
	}

	return ops, nil
}

// LoadSpec loads an OpenAPI 3.0/3.1 specification from disk.
func LoadSpec(filename string) (v3.Document, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return v3.Document{}, err
	}
	doc, err := libopenapi.NewDocumentWithConfiguration(b, &datamodel.DocumentConfiguration{
		AllowRemoteReferences: true,
	})
	if err != nil {
		return v3.Document{}, err
	}
	m, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		return v3.Document{}, errors.Join(errs...)
	}

	return m.Model, nil
}

// newOperationFromSpec creates a new operation from the loaded specification.
func newOperationFromSpec(method string, path string, params []*v3.Parameter, specOp v3.Operation) Operation {
	op := Operation{
		ID:          strcase.ToKebab(specOp.OperationId),
		Summary:     specOp.Summary,
		Description: Sanitize(specOp.Description),
		Method:      method,
		Path:        path,
		Deprecated:  getBool(specOp.Deprecated),
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
	// Parameters can be defined per-path or per-operation.
	for _, param := range params {
		op.Parameters.Add(newParameterFromSpec(*param))
	}
	for _, param := range specOp.Parameters {
		op.Parameters.Add(newParameterFromSpec(*param))
	}
	if specOp.RequestBody != nil && specOp.RequestBody.Content != nil {
		pair := orderedmap.First(specOp.RequestBody.Content)
		format := pair.Key()
		mediaType := pair.Value()
		mediaTypeSchema := mediaType.Schema.Schema()

		op.Parameters.Add(newBodyParameters("", mediaTypeSchema)...)
		op.BodyFormat = format
	}

	return op
}

// newParameterFromSpec creates a new parameter from the loaded specification.
func newParameterFromSpec(specParam v3.Parameter) Parameter {
	schema := specParam.Schema.Schema()

	return Parameter{
		In:          specParam.In,
		Name:        specParam.Name,
		Description: Sanitize(specParam.Description),
		Style:       specParam.Style,
		Type:        getSchemaType(schema),
		Enum:        getEnum(schema),
		Example:     getExample(schema),
		Default:     getDefaultValue(schema),
		Deprecated:  specParam.Deprecated,
		Required:    getBool(specParam.Required),
	}
}

// newBodyParameters creates a slice of body parameters from the given schema.
func newBodyParameters(prefix string, schema *base.Schema) []Parameter {
	parameters := make([]Parameter, 0, 10)
	for pair := orderedmap.First(schema.Properties); pair != nil; pair = pair.Next() {
		propertyName := pair.Key()
		propertySchema := pair.Value().Schema()
		propertySchemaType := getSchemaType(propertySchema)

		if propertySchemaType == "object" {
			// Nested parameters found, flatten them.
			parameters = append(parameters, newBodyParameters(prefix+propertyName+".", propertySchema)...)
		} else {
			parameters = append(parameters, Parameter{
				In:          "body",
				Name:        prefix + propertyName,
				Description: Sanitize(propertySchema.Description),
				Type:        propertySchemaType,
				Enum:        getEnum(propertySchema),
				Example:     getExample(propertySchema),
				Default:     getDefaultValue(propertySchema),
				Deprecated:  getBool(propertySchema.Deprecated),
				Required:    slices.Contains(schema.Required, propertyName),
			})
		}
	}

	return parameters
}

// getSchemaType retrieves the type of the given schema.
func getSchemaType(schema *base.Schema) string {
	// schema.Type can contain multiple values in OpenAPI 3.1, e.g:
	// [string, null] or [string, integer]. Broom needs a single type
	// so that it can cast the value (see Parameter#CastString).
	schemaType := schema.Type[0]
	// Expand the array type into the underlying type (array -> []string).
	if schemaType == "array" && schema.Items.IsA() {
		schemaType = fmt.Sprintf("[]%v", schema.Items.A.Schema().Type[0])
	}

	return schemaType
}

// getEnum retrieves the enum values defined on the given schema.
func getEnum(schema *base.Schema) []string {
	var enum []string
	for _, v := range schema.Enum {
		enum = append(enum, v.Value)
	}

	return enum
}

// getExample retrieves the examples defined on the given schema.
func getExample(schema *base.Schema) string {
	var exampleValue string
	if len(schema.Examples) > 0 {
		exampleValue = schema.Examples[0].Value
	} else if schema.Example != nil {
		exampleValue = schema.Example.Value
	}

	return exampleValue
}

// getDefaultValue retrieves the default value defined on the given schema.
func getDefaultValue(schema *base.Schema) string {
	if schema.Default == nil {
		return ""
	}
	return schema.Default.Value
}

// getBool converts a boolean reference into a boolean, turning nil into false.
func getBool(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}
