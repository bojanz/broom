// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

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

// Validate validates each parameter against the given values.
func (ps Parameters) Validate(values url.Values) error {
	for _, p := range ps {
		value := values.Get(p.Name)
		if err := p.Validate(value); err != nil {
			return err
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
	Enum        []string
	Deprecated  bool
	Required    bool
}

// Label returns a human-readable parameter label.
func (p Parameter) Label() string {
	return strings.Title(strcase.ToDelimited(p.Name, ' '))
}

// CastString casts the given string to the parameter type.
func (p Parameter) CastString(str string) (interface{}, error) {
	if strings.HasPrefix(p.Type, "[]") {
		strs := strings.Split(str, ",")
		vs := make([]interface{}, 0, len(strs))
		for _, s := range strs {
			v, err := parseStr(s, p.Type[2:])
			if err != nil {
				return nil, fmt.Errorf("%q is not a valid %v", s, p.Type[2:])
			}
			vs = append(vs, v)
		}
		return vs, nil
	} else {
		v, err := parseStr(str, p.Type)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid %v", str, p.Type)
		}
		return v, nil
	}
}

// parseStr invokes the strconv parse function for the given type.
func parseStr(str string, newType string) (interface{}, error) {
	if newType == "boolean" {
		return strconv.ParseBool(str)
	} else if newType == "integer" {
		return strconv.ParseInt(str, 10, 64)
	} else if newType == "number" {
		return strconv.ParseFloat(str, 64)
	}
	return str, nil
}

// Validate validates the parameter against the given value.
func (p Parameter) Validate(value string) error {
	if value == "" && p.Required {
		return fmt.Errorf("missing required %v parameter %q", p.In, p.Name)
	}
	// A strict check would not avoid empty strings, requiring them to be
	// declared in the enum as well, but since many specs don't do that,
	// the check here is loosened to prevent user frustration.
	if value != "" && len(p.Enum) > 0 && !contains(p.Enum, value) {
		formattedEnum := strings.Join(p.Enum, ", ")
		return fmt.Errorf("invalid value for %v parameter %q (allowed values: %v)", p.In, p.Name, formattedEnum)
	}

	return nil
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

// HasBody returns whether the operation has a body.
func (op Operation) HasBody() bool {
	// Body params are keyed by format in the spec, so there's no need to check both.
	return op.BodyFormat != ""
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
	if !op.HasBody() {
		// Operation does not support specifying a body (e.g. GET/DELETE).
		return nil, nil
	}
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
