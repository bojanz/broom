// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom_test

import (
	"net/url"
	"testing"

	"github.com/bojanz/broom"
	"github.com/google/go-cmp/cmp"
)

func TestOperations_ByID(t *testing.T) {
	operations := broom.Operations{
		broom.Operation{ID: "create-user"},
		broom.Operation{ID: "update-user"},
	}

	op1, ok := operations.ByID("update-user")
	if op1.ID != "update-user" || ok != true {
		t.Errorf("got %v, %v want update-user, true", op1.ID, ok)
	}

	op2, ok := operations.ByID("create-user")
	if op2.ID != "create-user" || ok != true {
		t.Errorf("got %v, %v want create-user, true", op2.ID, ok)
	}

	op3, ok := operations.ByID("delete-user")
	if op3.ID != "" || ok != false {
		t.Errorf(`got %v, %v want "", false`, op1.ID, ok)
	}
}

func TestOperations_ByTag(t *testing.T) {
	operations := broom.Operations{
		broom.Operation{ID: "create-product", Tag: "Products"},
		broom.Operation{ID: "update-product", Tag: "Products"},
		broom.Operation{ID: "delete-product", Tag: "Products"},
		broom.Operation{ID: "create-user", Tag: "Users"},
		broom.Operation{ID: "update-user", Tag: "Users"},
	}

	gotOps := operations.ByTag("Products")
	wantOps := broom.Operations{
		broom.Operation{ID: "create-product", Tag: "Products"},
		broom.Operation{ID: "update-product", Tag: "Products"},
		broom.Operation{ID: "delete-product", Tag: "Products"},
	}
	if diff := cmp.Diff(wantOps, gotOps); diff != "" {
		t.Errorf("product operation mismatch (-want +got):\n%s", diff)
	}

	gotOps = operations.ByTag("Users")
	wantOps = broom.Operations{
		broom.Operation{ID: "create-user", Tag: "Users"},
		broom.Operation{ID: "update-user", Tag: "Users"},
	}
	if diff := cmp.Diff(wantOps, gotOps); diff != "" {
		t.Errorf("user operation mismatch (-want +got):\n%s", diff)
	}
}

func TestOperations_Tags(t *testing.T) {
	operations := broom.Operations{
		broom.Operation{ID: "create-product", Tag: "Products"},
		broom.Operation{ID: "update-product", Tag: "Products"},
		broom.Operation{ID: "delete-product", Tag: "Products"},
		broom.Operation{ID: "create-user", Tag: "Users"},
		broom.Operation{ID: "update-user", Tag: "Users"},
	}

	wantTags := []string{"Products", "Users"}
	gotTags := operations.Tags()
	if !cmp.Equal(gotTags, wantTags) {
		t.Errorf("got %v, want %v", gotTags, wantTags)
	}
}

func TestParameters_ByName(t *testing.T) {
	parameters := broom.Parameters{
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

func TestParameters_Validate(t *testing.T) {
	parameters := broom.Parameters{
		broom.Parameter{
			In:       "query",
			Name:     "billing_country",
			Required: true,
		},
		broom.Parameter{
			In:   "query",
			Name: "sort",
			Enum: []string{"billing_country", "user_id"},
		},
	}

	// Required parameter missing.
	err := parameters.Validate(url.Values{})
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `missing required query parameter "billing_country"` {
		t.Errorf("unexpected error %v", err)
	}

	// Required parameter provided but empty
	err = parameters.Validate(url.Values{"billing_country": {""}})
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `missing required query parameter "billing_country"` {
		t.Errorf("unexpected error %v", err)
	}

	// Provided required parameter.
	err = parameters.Validate(url.Values{"billing_country": {"US"}})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Enum validation.
	err = parameters.Validate(url.Values{"billing_country": {"US"}, "sort": {"invalid"}})
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `invalid value for query parameter "sort" (allowed values: billing_country, user_id)` {
		t.Errorf("unexpected error %v", err)
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

func TestParameter_NameWithFlags(t *testing.T) {
	param := broom.Parameter{
		Name: "first_name",
	}
	got := param.NameWithFlags()
	want := "first_name"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	param = broom.Parameter{
		Name:     "first_name",
		Required: true,
	}
	got = param.NameWithFlags()
	want = "first_name (required)"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	param = broom.Parameter{
		Name:       "first_name",
		Deprecated: true,
		Required:   true,
	}
	got = param.NameWithFlags()
	want = "first_name (deprecated, required)"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestOperation_ParametersIn(t *testing.T) {
	operation := broom.Operation{
		Parameters: broom.Parameters{
			broom.Parameter{
				In:   "path",
				Name: "userId",
			},
			broom.Parameter{
				In:   "path",
				Name: "Id",
			},
			broom.Parameter{
				In:   "query",
				Name: "page",
			},
		},
	}

	gotParams := operation.ParametersIn("path")
	wantParams := broom.Parameters{
		broom.Parameter{
			In:   "path",
			Name: "userId",
		},
		broom.Parameter{
			In:   "path",
			Name: "Id",
		}}
	if diff := cmp.Diff(wantParams, gotParams); diff != "" {
		t.Errorf("path parameter mismatch (-want +got):\n%s", diff)
	}

	gotParams = operation.ParametersIn("query")
	wantParams = broom.Parameters{
		broom.Parameter{
			In:   "query",
			Name: "page",
		},
	}
	if diff := cmp.Diff(wantParams, gotParams); diff != "" {
		t.Errorf("query parameter mismatch (-want +got):\n%s", diff)
	}

	gotParams = operation.ParametersIn("body")
	wantParams = broom.Parameters{}
	if diff := cmp.Diff(wantParams, gotParams); diff != "" {
		t.Errorf("body parameter mismatch (-want +got):\n%s", diff)
	}
}

func TestOperation_ProcessBody(t *testing.T) {
	// Empty format.
	operation := broom.Operation{}
	b, err := operation.ProcessBody("username=jsmith")
	if len(b) != 0 {
		t.Errorf("expected an empty body, got %v", string(b))
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Unsupported format.
	operation = broom.Operation{
		Parameters: broom.Parameters{
			broom.Parameter{
				In:       "body",
				Name:     "username",
				Type:     "string",
				Required: true,
			},
		},
		BodyFormat: "application/xml",
	}
	b, err = operation.ProcessBody("username=jsmith")
	if len(b) != 0 {
		t.Errorf("expected an empty body, got %v", string(b))
	}
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != "unsupported body format application/xml" {
		t.Errorf("unexpected error %v", err)
	}

	// Missing required parameter.
	b, err = operation.ProcessBody("")
	if len(b) != 0 {
		t.Errorf("expected an empty body, got %v", string(b))
	}
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `missing required body parameter "username"` {
		t.Errorf("unexpected error %v", err)
	}

	// Form data (application/x-www-form-urlencoded).
	operation = broom.Operation{
		Parameters: broom.Parameters{
			broom.Parameter{
				In:   "body",
				Name: "username",
				Type: "string",
			},
		},
		BodyFormat: "application/x-www-form-urlencoded",
	}
	// Non-defined parameters are expected to be passed through.
	b, err = operation.ProcessBody("email=js@domain&username=jsmith")
	got := string(b)
	want := "email=js%40domain&username=jsmith"
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// JSON data (application/json).
	operation = broom.Operation{
		Parameters: broom.Parameters{
			// Parameters without a type should remain strings.
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
		},
		BodyFormat: "application/json",
	}
	// Invalid boolean.
	b, err = operation.ProcessBody("status=invalid")
	if len(b) != 0 {
		t.Errorf("expected an empty body, got %v", string(b))
	}
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `could not process status: "invalid" is not a valid boolean` {
		t.Errorf("unexpected error %v", err)
	}

	// Invalid integer.
	b, err = operation.ProcessBody("storage=3.2")
	if len(b) != 0 {
		t.Errorf("expected an empty body, got %v", string(b))
	}
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `could not process storage: "3.2" is not a valid integer` {
		t.Errorf("unexpected error %v", err)
	}

	// Invalid integer in an array.
	b, err = operation.ProcessBody("lucky_numbers=4,eight,15")
	if len(b) != 0 {
		t.Errorf("expected an empty body, got %v", string(b))
	}
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `could not process lucky_numbers: "eight" is not a valid integer` {
		t.Errorf("unexpected error %v", err)
	}

	// Invalid number.
	b, err = operation.ProcessBody("vcpu=1,7")
	if len(b) != 0 {
		t.Errorf("expected an empty body, got %v", string(b))
	}
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `could not process vcpu: "1,7" is not a valid number` {
		t.Errorf("unexpected error %v", err)
	}

	// Valid data.
	b, err = operation.ProcessBody("email=js@domain&lucky_numbers=4,8,15,16,23,42&username=jsmith&roles=admin,owner&storage=20480&vcpu=0.5&status=true")
	got = string(b)
	// Note: keys are always alphabetical, due to how encoding/json treats maps.
	want = `{"email":"js@domain","lucky_numbers":[4,8,15,16,23,42],"roles":["admin","owner"],"status":true,"storage":20480,"username":"jsmith","vcpu":0.5}`
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestOperation_RealPath(t *testing.T) {
	// No path parameters.
	operation := broom.Operation{
		Path: "/users",
	}
	path, err := operation.RealPath(nil, "")
	if path != "/users" {
		t.Errorf("got %v, want /users", path)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// No path parameters, but one provided anyway.
	path, err = operation.RealPath([]string{"ignore-me"}, "")
	if path != "/users" {
		t.Errorf("got %v, want /users", path)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Missing path parameter.
	operation = broom.Operation{
		Path: "/users/{userId}",
		Parameters: broom.Parameters{
			broom.Parameter{
				In:   "path",
				Name: "userId",
			},
		},
	}
	path, err = operation.RealPath(nil, "")
	if path != "" {
		t.Errorf("unexpected path %v", path)
	}
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != "too few path parameters: got 0, want 1" {
		t.Errorf("unexpected error %v", err)
	}

	// Provided path parameters.
	operation = broom.Operation{
		Path: "/users/{userId}/orders/{orderId}",
		Parameters: broom.Parameters{
			broom.Parameter{
				In:   "path",
				Name: "userId",
			},
			broom.Parameter{
				In:   "path",
				Name: "orderId",
			},
		},
	}
	path, err = operation.RealPath([]string{"test-user", "123456"}, "")
	if path != "/users/test-user/orders/123456" {
		t.Errorf("got %v, want /users/test-user/orders/123456", path)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Path and query parameters.
	operation = broom.Operation{
		Path: "/users/{userId}/orders",
		Parameters: broom.Parameters{
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
		},
	}
	path, err = operation.RealPath([]string{"test-user"}, "sort=-updated_at")
	if path != "" {
		t.Errorf("unexpected path %v", path)
	}
	if err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != `missing required query parameter "billing_country"` {
		t.Errorf("unexpected error %v", err)
	}

	// One required, one non-defined query parameter.
	path, err = operation.RealPath([]string{"test-user"}, "billing_country=US&billing_region=NY&sort=-updated_at")
	if path != "/users/test-user/orders?billing_country=US&billing_region=NY&sort=-updated_at" {
		t.Errorf("got %v, want /users/test-user/orders?billing_country=US&billing_region=NY&sort=-updated_at", path)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Confirm that query parameters are escaped.
	path, err = operation.RealPath([]string{"test-user"}, "billing_country=U S")
	if path != "/users/test-user/orders?billing_country=U+S" {
		t.Errorf("got %v, want /users/test-user/orders?billing_country=U+S", path)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}
