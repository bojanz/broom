// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bojanz/broom"
	"github.com/google/go-cmp/cmp"
)

func TestOperations_ByID(t *testing.T) {
	ops := broom.Operations{
		broom.Operation{ID: "create-user"},
		broom.Operation{ID: "update-user"},
	}

	op1, ok := ops.ByID("update-user")
	if op1.ID != "update-user" || ok != true {
		t.Errorf("got %v, %v want update-user, true", op1.ID, ok)
	}

	op2, ok := ops.ByID("create-user")
	if op2.ID != "create-user" || ok != true {
		t.Errorf("got %v, %v want create-user, true", op2.ID, ok)
	}

	op3, ok := ops.ByID("delete-user")
	if op3.ID != "" || ok != false {
		t.Errorf(`got %v, %v want "", false`, op1.ID, ok)
	}
}

func TestOperations_ByTag(t *testing.T) {
	ops := broom.Operations{
		broom.Operation{ID: "create-product", Tag: "Products"},
		broom.Operation{ID: "update-product", Tag: "Products"},
		broom.Operation{ID: "delete-product", Tag: "Products"},
		broom.Operation{ID: "create-user", Tag: "Users"},
		broom.Operation{ID: "update-user", Tag: "Users"},
	}

	gotOps := ops.ByTag("Products")
	wantOps := broom.Operations{
		broom.Operation{ID: "create-product", Tag: "Products"},
		broom.Operation{ID: "update-product", Tag: "Products"},
		broom.Operation{ID: "delete-product", Tag: "Products"},
	}
	if diff := cmp.Diff(wantOps, gotOps); diff != "" {
		t.Errorf("product operation mismatch (-want +got):\n%s", diff)
	}

	gotOps = ops.ByTag("Users")
	wantOps = broom.Operations{
		broom.Operation{ID: "create-user", Tag: "Users"},
		broom.Operation{ID: "update-user", Tag: "Users"},
	}
	if diff := cmp.Diff(wantOps, gotOps); diff != "" {
		t.Errorf("user operation mismatch (-want +got):\n%s", diff)
	}
}

func TestOperations_Tags(t *testing.T) {
	ops := broom.Operations{
		broom.Operation{ID: "create-product", Tag: "Products"},
		broom.Operation{ID: "update-product", Tag: "Products"},
		broom.Operation{ID: "delete-product", Tag: "Products"},
		broom.Operation{ID: "create-user", Tag: "Users"},
		broom.Operation{ID: "update-user", Tag: "Users"},
	}

	wantTags := []string{"Products", "Users"}
	gotTags := ops.Tags()
	if !cmp.Equal(gotTags, wantTags) {
		t.Errorf("got %v, want %v", gotTags, wantTags)
	}
}

func TestOperation_SummaryWithFlags(t *testing.T) {
	op := broom.Operation{
		Summary: "List products",
	}
	got := op.SummaryWithFlags()
	want := "List products"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	op = broom.Operation{
		Summary:    "List products",
		Deprecated: true,
	}
	got = op.SummaryWithFlags()
	want = "List products (deprecated)"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestOperation_Validate(t *testing.T) {
	// Missing path parameter.
	op := broom.Operation{Path: "/users/{userId}"}
	op.Parameters.Add(broom.Parameter{
		In:   "path",
		Name: "userId",
	})
	err := op.Validate(broom.RequestValues{})
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != "too few path parameters: got 0, want 1" {
		t.Errorf("unexpected error %v", err)
	}
}

func TestOperation_Request(t *testing.T) {
	// No path parameters. Server URL with trailing slash.
	op := broom.Operation{Method: "GET", Path: "/users"}
	req, err := op.Request("https://myapi.io/", broom.RequestValues{})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("got %v, want GET", req.Method)
	}
	if req.URL.String() != "https://myapi.io/users" {
		t.Errorf("got %v, want https://myapi.io/users", req.URL.String())
	}

	// No path parameters, but one provided anyway.
	values, _ := broom.ParseRequestValues(nil, []string{"ignore-me"}, "", "")
	req, err = op.Request("https://myapi.io", values)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("got %v, want GET", req.Method)
	}
	if req.URL.String() != "https://myapi.io/users" {
		t.Errorf("got %v, want https://myapi.io/users", req.URL.String())
	}

	// Provided path parameters.
	op = broom.Operation{Method: "GET", Path: "/users/{userId}/orders/{orderId}"}
	op.Parameters.Add(
		broom.Parameter{
			In:   "path",
			Name: "userId",
		},
		broom.Parameter{
			In:   "path",
			Name: "orderId",
		},
	)
	values, _ = broom.ParseRequestValues(nil, []string{"test-user", "123456"}, "", "")
	req, err = op.Request("https://myapi.io", values)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("got %v, want GET", req.Method)
	}
	if req.URL.String() != "https://myapi.io/users/test-user/orders/123456" {
		t.Errorf("got %v, want https://myapi.io/users/test-user/orders/123456", req.URL.String())
	}

	// One required, one non-defined query parameter.
	op = broom.Operation{Method: "GET", Path: "/users/{userId}/orders"}
	op.Parameters.Add(
		broom.Parameter{
			In:   "path",
			Name: "userId",
		},
		broom.Parameter{
			In:       "query",
			Name:     "billing_country",
			Required: true,
		},
		broom.Parameter{
			In:   "query",
			Name: "sort",
		},
	)
	values, _ = broom.ParseRequestValues(nil, []string{"test-user"}, "billing_country=US&billing_region=NY&sort=-updated_at", "")
	req, err = op.Request("https://myapi.io", values)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("got %v, want GET", req.Method)
	}
	if req.URL.String() != "https://myapi.io/users/test-user/orders?billing_country=US&billing_region=NY&sort=-updated_at" {
		t.Errorf("got %v, want https://myapi.io/users/test-user/orders?billing_country=US&billing_region=NY&sort=-updated_at", req.URL.String())
	}

	// Confirm that query parameters are escaped.
	values, _ = broom.ParseRequestValues(nil, []string{"test-user"}, "billing_country=U S", "")
	req, err = op.Request("https://myapi.io", values)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("got %v, want GET", req.Method)
	}
	if req.URL.String() != "https://myapi.io/users/test-user/orders?billing_country=U+S" {
		t.Errorf("got %v, want https://myapi.io/users/test-user/orders?billing_country=U+S", req.URL.String())
	}

	// Confirm that the expected headers are set.
	op = broom.Operation{Method: "GET", Path: "/users"}
	values, _ = broom.ParseRequestValues([]string{"X-Vendor: Test"}, nil, "", "")
	req, err = op.Request("https://myapi.io", values)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if req.Method != "GET" {
		t.Errorf("got %v, want GET", req.Method)
	}
	if req.URL.String() != "https://myapi.io/users" {
		t.Errorf("got %v, want https://myapi.io/users", req.URL.String())
	}
	if req.Header.Get("X-Vendor") != "Test" {
		t.Errorf("got %v, want Test", req.Header.Get("X-Vendor"))
	}
	if ua := req.Header.Get("User-Agent"); !strings.HasPrefix(ua, "broom") {
		t.Errorf("unexpected user agent %v", req.Header.Get("User-Agent"))
	}
}

func TestOperation_RequestWithBody(t *testing.T) {
	// Empty format.
	op := broom.Operation{Method: "POST", Path: "/users"}
	values, _ := broom.ParseRequestValues(nil, nil, "", "username=jsmith")
	req, err := op.Request("https://myapi.io", values)
	b, _ := io.ReadAll(req.Body)
	if len(b) != 0 {
		t.Errorf("expected an empty body, got %v", string(b))
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Unsupported format.
	op = broom.Operation{
		Method:     "POST",
		Path:       "/users",
		BodyFormat: "application/xml",
	}
	op.Parameters.Add(
		broom.Parameter{
			In:       "body",
			Name:     "username",
			Type:     "string",
			Required: true,
		},
	)
	_, err = op.Request("https://myapi.io", values)
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != "unsupported body format application/xml" {
		t.Errorf("unexpected error %v", err)
	}

	// Form data (application/x-www-form-urlencoded).
	op = broom.Operation{
		Method:     "POST",
		Path:       "/users",
		BodyFormat: "application/x-www-form-urlencoded",
	}
	op.Parameters.Add(
		broom.Parameter{
			In:   "body",
			Name: "username",
			Type: "string",
		},
	)
	// Non-defined parameters are expected to be passed through.
	values, _ = broom.ParseRequestValues(nil, nil, "", "email=js@domain&username=jsmith")
	req, err = op.Request("https://myapi.io", values)
	b, _ = io.ReadAll(req.Body)
	got := string(b)
	want := "email=js%40domain&username=jsmith"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("got %v, want POST", req.Method)
	}
	if req.URL.String() != "https://myapi.io/users" {
		t.Errorf("got %v, want https://myapi.io/users", req.URL.String())
	}
	if req.Header.Get("Content-Type") != op.BodyFormat {
		t.Errorf("got %v, want %v", req.Header.Get("Content-Type"), op.BodyFormat)
	}

	// JSON data (application/json).
	// Parameters without a type should remain strings.
	op = broom.Operation{
		Method:     "POST",
		Path:       "/users",
		BodyFormat: "application/json",
	}
	op.Parameters.Add(
		broom.Parameter{
			In:   "body",
			Name: "username",
		},
		broom.Parameter{
			In:   "body",
			Name: "roles",
			Type: "[]string",
		},
		broom.Parameter{
			In:   "body",
			Name: "lucky_numbers",
			Type: "[]integer",
		},
		broom.Parameter{
			In:   "body",
			Name: "storage",
			Type: "integer",
		},
		broom.Parameter{
			In:   "body",
			Name: "vcpu",
			Type: "number",
		},
		broom.Parameter{
			In:   "body",
			Name: "status",
			Type: "boolean",
		},
	)
	// Invalid boolean.
	values, _ = broom.ParseRequestValues(nil, nil, "", "status=invalid")
	_, err = op.Request("https://myapi.io", values)
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `could not process status: "invalid" is not a valid boolean` {
		t.Errorf("unexpected error %v", err)
	}

	// Invalid integer.
	values, _ = broom.ParseRequestValues(nil, nil, "", "storage=3.2")
	_, err = op.Request("https://myapi.io", values)
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `could not process storage: "3.2" is not a valid integer` {
		t.Errorf("unexpected error %v", err)
	}

	// Invalid integer in an array.
	values, _ = broom.ParseRequestValues(nil, nil, "", "lucky_numbers=4,eight,15")
	_, err = op.Request("https://myapi.io", values)
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `could not process lucky_numbers: "eight" is not a valid integer` {
		t.Errorf("unexpected error %v", err)
	}

	// Invalid number.
	values, _ = broom.ParseRequestValues(nil, nil, "", "vcpu=1,7")
	_, err = op.Request("https://myapi.io", values)
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `could not process vcpu: "1,7" is not a valid number` {
		t.Errorf("unexpected error %v", err)
	}

	// Valid data.
	values, _ = broom.ParseRequestValues(nil, nil, "", "email=js@domain&lucky_numbers=4,8,15,16,23,42&username=jsmith&roles=admin,owner&storage=20480&vcpu=0.5&status=true")
	req, err = op.Request("https://myapi.io", values)
	b, _ = io.ReadAll(req.Body)
	got = string(b)
	// Note: keys are always alphabetical, due to how encoding/json treats maps.
	want = `{"email":"js@domain","lucky_numbers":[4,8,15,16,23,42],"roles":["admin","owner"],"status":true,"storage":20480,"username":"jsmith","vcpu":0.5}`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("got %v, want POST", req.Method)
	}
	if req.URL.String() != "https://myapi.io/users" {
		t.Errorf("got %v, want https://myapi.io/users", req.URL.String())
	}
	if req.Header.Get("Content-Type") != op.BodyFormat {
		t.Errorf("got %v, want %v", req.Header.Get("Content-Type"), op.BodyFormat)
	}
}

func TestParameters_ByName(t *testing.T) {
	parameters := broom.ParameterList{
		broom.Parameter{
			In:       "query",
			Name:     "billing_country",
			Required: true,
		},
		broom.Parameter{
			In:   "query",
			Name: "sort",
		},
	}

	p1, ok := parameters.ByName("billing_country")
	if p1.Name != "billing_country" || ok != true {
		t.Errorf("got %v, %v want billing_country, true", p1.Name, ok)
	}

	p2, ok := parameters.ByName("sort")
	if p2.Name != "sort" || ok != true {
		t.Errorf("got %v, %v want sort, true", p2.Name, ok)
	}

	p3, ok := parameters.ByName("billing_region")
	if p3.Name != "" || ok != false {
		t.Errorf(`got %v, %v want "", false`, p3.Name, ok)
	}
}

func TestParameter_Label(t *testing.T) {
	// Conversion from snake_case.
	param := broom.Parameter{
		Name: "first_name",
	}
	got := param.Label()
	want := "First Name"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Conversion from camelCase.
	param = broom.Parameter{
		Name: "lastName",
	}
	got = param.Label()
	want = "Last Name"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParameter_FormattedFlags(t *testing.T) {
	param := broom.Parameter{
		Name: "first_name",
	}
	got := param.FormattedFlags()
	want := ""
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	param = broom.Parameter{
		Name:     "first_name",
		Required: true,
	}
	got = param.FormattedFlags()
	want = "(required)"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	param = broom.Parameter{
		Name:       "first_name",
		Deprecated: true,
		Required:   true,
	}
	got = param.FormattedFlags()
	want = "(deprecated, required)"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseRequestValues(t *testing.T) {
	// Invalid header.
	_, err := broom.ParseRequestValues([]string{"X-Vendor Test"}, nil, "", "")
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `parse header: could not parse "X-Vendor Test"` {
		t.Errorf("unexpected error %v", err)
	}

	// Invalid query.
	_, err = broom.ParseRequestValues(nil, nil, "first_name=john;last_name=smith", "")
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `parse query: invalid semicolon separator in query` {
		t.Errorf("unexpected error %v", err)
	}

	// Invalid body.
	_, err = broom.ParseRequestValues(nil, nil, "", "first_name=john;last_name=smith")
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `parse body: invalid semicolon separator in query` {
		t.Errorf("unexpected error %v", err)
	}

	// Valid values.
	values, err := broom.ParseRequestValues([]string{"X-Vendor:Test"}, []string{"a", "b"}, "filter[deleted]=true", "first_name=john&last_name=smith")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	wantHeader := http.Header{}
	wantHeader.Add("X-Vendor", "Test")
	if !cmp.Equal(values.Header, wantHeader) {
		t.Errorf("got %v, want %v", values.Header, wantHeader)
	}
	wantPath := []string{"a", "b"}
	if !cmp.Equal(values.Path, wantPath) {
		t.Errorf("got %v, want %v", values.Path, wantPath)
	}
	wantQuery := url.Values{}
	wantQuery.Add("filter[deleted]", "true")
	if !cmp.Equal(values.Query, wantQuery) {
		t.Errorf("got %v, want %v", values.Query, wantQuery)
	}
	wantBody := url.Values{}
	wantBody.Add("first_name", "john")
	wantBody.Add("last_name", "smith")
	if !cmp.Equal(values.Body, wantBody) {
		t.Errorf("got %v, want %v", values.Body, wantBody)
	}
}
