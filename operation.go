package broom

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/iancoleman/strcase"
)

// Operations represents a list of operations.
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
func (ops Operations) ByTag(tag string) Operations {
	filteredOps := make(Operations, 0, len(ops))
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

// Parameters represents a list of parameters.
type Parameters []Parameter

// ByName returns a parameter with the given name.
func (ps Parameters) ByName(name string) (Parameter, bool) {
	for _, p := range ps {
		if p.Name == name {
			return p, true
		}
	}
	return Parameter{}, false
}

// Validate confirms the presence of required parameters in the given values.
func (ps Parameters) Validate(values url.Values) error {
	for _, p := range ps {
		if p.Required && values.Get(p.Name) == "" {
			return fmt.Errorf("missing required %v parameter %q", p.In, p.Name)
		}
	}
	return nil
}

// Parameter represents an operation parameter.
type Parameter struct {
	In          string
	Name        string
	Description string
	Style       string
	Type        string
	Deprecated  bool
	Required    bool
}

// CastString casts the given string to the parameter type.
func (p Parameter) CastString(str string) (interface{}, error) {
	switch p.Type {
	case "array":
		// @todo Support non-string arrays.
		return strings.Split(str, ","), nil
	case "boolean":
		v, err := strconv.ParseBool(str)
		if err != nil {
			return false, fmt.Errorf("%q is not a valid boolean", str)
		}
		return v, nil
	case "integer":
		v, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return false, fmt.Errorf("%q is not a valid integer", str)
		}
		return v, nil
	case "number":
		v, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return false, fmt.Errorf("%q is not a valid decimal number", str)
		}
		return v, nil
	}

	return str, nil
}

// Operation represents an available operation.
type Operation struct {
	ID          string
	Summary     string
	Description string
	Tag         string
	Method      string
	Path        string
	Parameters  Parameters
	BodyFormat  string
	Deprecated  bool
}

// ParametersIn returns a list of parameters in the given location (query, path, body).
func (op Operation) ParametersIn(in string) Parameters {
	filteredParams := make(Parameters, 0, len(op.Parameters))
	for _, param := range op.Parameters {
		if param.In == in {
			filteredParams = append(filteredParams, param)
		}
	}
	return filteredParams
}

// ProcessBody converts the given body string into a byte array suitable for sending.
func (op Operation) ProcessBody(body string) ([]byte, error) {
	values, err := url.ParseQuery(body)
	if err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}
	bodyParams := op.ParametersIn("body")
	if err := bodyParams.Validate(values); err != nil {
		return nil, err
	}

	if IsJSON(op.BodyFormat) {
		jsonValues := make(map[string]interface{}, len(values))
		for name := range values {
			value := values.Get(name)
			// Allow defined parameters to cast the string.
			if bodyParam, ok := bodyParams.ByName(name); ok {
				jsonValues[name], err = bodyParam.CastString(value)
				if err != nil {
					return nil, fmt.Errorf("could not process %v: %v", name, err)
				}
			}
			// Pass through non-defined parameters as strings.
			if _, ok := jsonValues[name]; !ok {
				jsonValues[name] = value
			}
		}
		return json.Marshal(jsonValues)
	} else if op.BodyFormat == "application/x-www-form-urlencoded" {
		return []byte(values.Encode()), nil
	} else {
		return nil, fmt.Errorf("unsupported body format %v", op.BodyFormat)
	}
}

// RealPath returns a path with the given path and query parameters included.
func (op Operation) RealPath(pathValues []string, query string) (string, error) {
	pathParams := op.ParametersIn("path")
	nParams := len(pathParams)
	nValues := len(pathValues)
	if nParams > nValues {
		return "", fmt.Errorf("too few path parameters: got %v, want %v", nValues, nParams)
	}
	replace := make([]string, 0, len(pathParams)*2)
	for i, param := range pathParams {
		paramName := fmt.Sprintf("{%v}", param.Name)
		replace = append(replace, paramName, pathValues[i])
	}
	r := strings.NewReplacer(replace...)
	path := r.Replace(op.Path)
	if query != "" {
		queryValues, err := url.ParseQuery(query)
		if err != nil {
			return "", fmt.Errorf("parse query: %w", err)
		}
		queryParams := op.ParametersIn("query")
		if err := queryParams.Validate(queryValues); err != nil {
			return "", err
		}
		path = path + "?" + queryValues.Encode()
	}

	return path, nil
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
	op.Parameters = make(Parameters, 0, len(params)+len(specOp.Parameters))
	for _, param := range params {
		op.Parameters = append(op.Parameters, NewParameterFromSpec(*param.Value))
	}
	for _, param := range specOp.Parameters {
		op.Parameters = append(op.Parameters, NewParameterFromSpec(*param.Value))
	}
	if specOp.RequestBody != nil {
		for format, mediaType := range specOp.RequestBody.Value.Content {
			op.BodyFormat = format
			for name, schema := range mediaType.Schema.Value.Properties {
				required := false
				for _, requiredName := range mediaType.Schema.Value.Required {
					if requiredName == name {
						required = true
					}
				}

				op.Parameters = append(op.Parameters, Parameter{
					In:          "body",
					Name:        name,
					Description: schema.Value.Description,
					Type:        schema.Value.Type,
					Deprecated:  schema.Value.Deprecated,
					Required:    required,
				})
			}
			break
		}
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
		Type:        specParam.Schema.Value.Type,
		Deprecated:  specParam.Deprecated,
		Required:    specParam.Required,
	}
}
