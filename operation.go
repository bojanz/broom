// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
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

// SummaryWithFlags returns the operation summary with flags.
func (op Operation) SummaryWithFlags() string {
	summary := op.Summary
	if op.Deprecated {
		summary = fmt.Sprintf("%v (deprecated)", summary)
	}

	return summary
}

// HasBody returns whether the operation has a body.
func (op Operation) HasBody() bool {
	// Body params are keyed by format in the spec, so there's no need to check both.
	return op.BodyFormat != ""
}

// Validate validates the given values against the operation's parameters.
func (op Operation) Validate(values RequestValues) error {
	nParams := len(op.Parameters.Path)
	nValues := len(values.Path)
	if nParams > nValues {
		return fmt.Errorf("too few path parameters: got %v, want %v", nValues, nParams)
	}
	if err := op.Parameters.Query.Validate(values.Query); err != nil {
		return err
	}
	if err := op.Parameters.Body.Validate(values.Body); err != nil {
		return err
	}

	return nil
}

// Request creates a new request with the given values.
func (op Operation) Request(serverURL string, values RequestValues) (*http.Request, error) {
	if err := op.Validate(values); err != nil {
		return nil, err
	}
	url := op.requestURL(serverURL, values)
	body, err := op.requestBody(values.Body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(op.Method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if len(values.Header) > 0 {
		req.Header = values.Header
	}
	if op.HasBody() {
		req.Header.Set("Content-Type", op.BodyFormat)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("broom/%s (%s %s)", Version, runtime.GOOS, runtime.GOARCH))

	return req, nil
}

// requestURL builds an absolute request url using the given body/query values.
func (op Operation) requestURL(serverURL string, values RequestValues) string {
	oldnew := make([]string, 0, len(op.Parameters.Path)*2)
	for i, v := range values.Path {
		if i+1 > len(op.Parameters.Path) {
			break
		}
		paramName := fmt.Sprintf("{%v}", op.Parameters.Path[i].Name)
		oldnew = append(oldnew, paramName, v)
	}
	r := strings.NewReplacer(oldnew...)
	path := r.Replace(op.Path)
	if len(values.Query) > 0 {
		path = path + "?" + values.Query.Encode()
	}

	return serverURL + path
}

// requestBody converts the given body values into a byte array suitable for sending.
func (op Operation) requestBody(bodyValues url.Values) ([]byte, error) {
	if !op.HasBody() {
		// Operation does not support specifying a body (e.g. GET/DELETE).
		return nil, nil
	}

	if IsJSON(op.BodyFormat) {
		jsonValues := make(map[string]interface{}, len(bodyValues))
		for name := range bodyValues {
			value := bodyValues.Get(name)
			// Allow defined parameters to cast the string.
			if bodyParam, ok := op.Parameters.Body.ByName(name); ok {
				var err error
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
		return []byte(bodyValues.Encode()), nil
	} else {
		return nil, fmt.Errorf("unsupported body format %v", op.BodyFormat)
	}
}

// Parameters represents the operation's parameters.
type Parameters struct {
	Header ParameterList
	Path   ParameterList
	Query  ParameterList
	Body   ParameterList
}

// Add adds one or more parameters to the appropriate parameter list.
func (ps *Parameters) Add(params ...Parameter) {
	for _, p := range params {
		switch p.In {
		case "header":
			ps.Header = append(ps.Header, p)
		case "path":
			ps.Path = append(ps.Path, p)
		case "query":
			ps.Query = append(ps.Query, p)
		case "body":
			ps.Body = append(ps.Body, p)
		}
	}
}

// ParameterList represents a list of parameters.
type ParameterList []Parameter

// ByName returns a parameter with the given name.
func (pl ParameterList) ByName(name string) (Parameter, bool) {
	for _, p := range pl {
		if p.Name == name {
			return p, true
		}
	}
	return Parameter{}, false
}

// Validate validates each parameter against the given values.
func (pl ParameterList) Validate(values url.Values) error {
	for _, p := range pl {
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
	Default     interface{}
	Deprecated  bool
	Required    bool
}

// Label returns a human-readable parameter label.
func (p Parameter) Label() string {
	return strings.Title(strcase.ToDelimited(p.Name, ' '))
}

// NameWithFlags returns the parameter name with flags (deprecated, required)
func (p Parameter) NameWithFlags() string {
	flags := make([]string, 0, 2)
	if p.Deprecated {
		flags = append(flags, "deprecated")
	}
	if p.Required {
		flags = append(flags, "required")
	}
	name := p.Name
	if len(flags) > 0 {
		name = fmt.Sprintf("%v (%v)", name, strings.Join(flags, ", "))
	}

	return name
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

// RequestValues represent the values used to populate an operation request.
//
// Header, query, and body values are added to the request even if they don't
// have matching parameters, unlike path values, where the parameter is used
// to determine the name of the placeholder to replace.
type RequestValues struct {
	Header http.Header
	Path   []string
	Query  url.Values
	Body   url.Values
}

// ParseRequestValues parses parameter values from the given strings.
func ParseRequestValues(headers []string, pathValues []string, query string, body string) (RequestValues, error) {
	headerValues := make(http.Header, len(headers))
	for _, header := range headers {
		kv := strings.SplitN(header, ":", 2)
		if len(kv) < 2 {
			return RequestValues{}, fmt.Errorf("parse header: could not parse %q", header)
		}
		headerValues.Set(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
	}
	queryValues, err := url.ParseQuery(query)
	if err != nil {
		return RequestValues{}, fmt.Errorf("parse query: %w", err)
	}
	bodyValues, err := url.ParseQuery(body)
	if err != nil {
		return RequestValues{}, fmt.Errorf("parse body: %w", err)
	}
	values := RequestValues{
		Header: headerValues,
		Path:   pathValues,
		Query:  queryValues,
		Body:   bodyValues,
	}

	return values, nil
}
