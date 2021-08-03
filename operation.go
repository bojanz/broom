package broom

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/adler32"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/iancoleman/strcase"
)

// Operations represents all available operations.
type Operations []Operation

// ByID returns an operation with the given ID.
func (ops Operations) ByID(id string) (Operation, bool) {
	// Having to iterate over all operations is not super performant,
	// but it's something we only do once per command, it allows us
	// to avoid the problem of maps not being ordered.
	for _, op := range ops {
		if op.ID == id {
			return op, true
		}
	}
	return Operation{}, false
}

// ByTag returns a list of operations for the given tag.
func (ops Operations) ByTag(tag string) []Operation {
	filteredOps := make([]Operation, 0, len(ops))
	for _, op := range ops {
		if op.Tag == tag {
			filteredOps = append(filteredOps, op)
		}
	}
	return filteredOps
}

// Tags returns a list of all available operation tags.
func (ops Operations) Tags() []string {
	tags := make(map[string]struct{})
	for _, op := range ops {
		tags[op.Tag] = struct{}{}
	}
	tagNames := make([]string, 0, len(tags))
	for tagName := range tags {
		tagNames = append(tagNames, tagName)
	}
	sort.Strings(tagNames)

	return tagNames
}

// Parameter represents an operation parameter.
type Parameter struct {
	In          string
	Name        string
	Description string
	Style       string
	Deprecated  bool
	Required    bool
}

// Operation represents an available operation.
type Operation struct {
	ID          string
	Summary     string
	Description string
	Tag         string
	Method      string
	Path        string
	Parameters  []Parameter
	Deprecated  bool
}

// ParametersIn returns a list of parameters in the given location (query, path, header).
func (op Operation) ParametersIn(in string) []Parameter {
	filteredParams := make([]Parameter, 0, len(op.Parameters))
	for _, param := range op.Parameters {
		if param.In == in {
			filteredParams = append(filteredParams, param)
		}
	}
	return filteredParams
}

// RealPath returns a path with the given path and query parameters included.
func (op Operation) RealPath(pathValues []string, query string) (string, error) {
	pathParams := op.ParametersIn("path")
	nParams := len(pathParams)
	nValues := len(pathValues)
	if nParams > 0 && nParams > nValues {
		return "", fmt.Errorf("path has %v params, but only %v values were given", nParams, nValues)
	}
	replace := make([]string, 0, len(pathParams)*2)
	for i, param := range pathParams {
		paramName := fmt.Sprintf("{%v}", param.Name)
		replace = append(replace, paramName, pathValues[i])
	}
	r := strings.NewReplacer(replace...)
	path := r.Replace(op.Path)
	if query != "" {
		err := op.ValidateQuery(query)
		if err != nil {
			return "", err
		}
		path = path + "?" + query
	}

	return path, nil
}

// ValidateQuery validates the given query string against the defined parameters.
//
// Confirms that the query string is valid and that the required parameters are present.
// Allows the query string to have additional, non-defined parameters.
func (op Operation) ValidateQuery(query string) error {
	queryValues, err := url.ParseQuery(query)
	if err != nil {
		return fmt.Errorf("parse query: %w", err)
	}
	for _, param := range op.ParametersIn("query") {
		if param.Required && queryValues.Get(param.Name) == "" {
			return fmt.Errorf("missing required query parameter %q", param.Name)
		}
	}

	return nil
}

// LoadOperations loads available operations from the specified specification.
func LoadOperations(filename string) (Operations, error) {
	openapi3.DefineStringFormat("uuid", openapi3.FormatOfStringForUUIDOfRFC4122)
	openapi3.DefineStringFormat("ulid", `^[0-7]{1}[0-9A-HJKMNP-TV-Z]{25}$`)
	openapi3.SchemaFormatValidationDisabled = true

	spec, err := openapi3.NewLoader().LoadFromFile(filename)
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
			operation := NewOperationFromSpec(http.MethodGet, path, pathItem.Parameters, *pathItem.Get)
			operations = append(operations, operation)
		}
		if pathItem.Post != nil {
			operation := NewOperationFromSpec(http.MethodPost, path, pathItem.Parameters, *pathItem.Post)
			operations = append(operations, operation)
		}
		if pathItem.Put != nil {
			operation := NewOperationFromSpec(http.MethodPut, path, pathItem.Parameters, *pathItem.Put)
			operations = append(operations, operation)
		}
		if pathItem.Patch != nil {
			operation := NewOperationFromSpec(http.MethodPatch, path, pathItem.Parameters, *pathItem.Patch)
			operations = append(operations, operation)
		}
		if pathItem.Delete != nil {
			operation := NewOperationFromSpec(http.MethodDelete, path, pathItem.Parameters, *pathItem.Delete)
			operations = append(operations, operation)
		}
	}

	return operations, nil
}

// NewOperationFromSpec creates a new operation from the loaded specification.
func NewOperationFromSpec(method string, path string, params openapi3.Parameters, specOp openapi3.Operation) Operation {
	op := Operation{
		ID:          strcase.ToKebab(specOp.OperationID),
		Summary:     specOp.Summary,
		Description: specOp.Description,
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
	op.Parameters = make([]Parameter, 0, len(params)+len(specOp.Parameters))
	for _, param := range params {
		op.Parameters = append(op.Parameters, NewParameterFromSpec(*param.Value))
	}
	for _, param := range specOp.Parameters {
		op.Parameters = append(op.Parameters, NewParameterFromSpec(*param.Value))
	}

	return op
}

// NewParameterFromSpec creates a new parameter from the loaded specification.
func NewParameterFromSpec(specParam openapi3.Parameter) Parameter {
	return Parameter{
		In:          specParam.In,
		Name:        specParam.Name,
		Description: specParam.Description,
		Style:       specParam.Style,
		Deprecated:  specParam.Deprecated,
		Required:    specParam.Required,
	}
}
