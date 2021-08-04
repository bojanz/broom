package broom_test

import (
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

func TestOperation_ParametersIn(t *testing.T) {
	operation := broom.Operation{
		Parameters: []broom.Parameter{
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
	wantParams := []broom.Parameter{
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
	wantParams = []broom.Parameter{
		broom.Parameter{
			In:   "query",
			Name: "page",
		},
	}
	if diff := cmp.Diff(wantParams, gotParams); diff != "" {
		t.Errorf("query parameter mismatch (-want +got):\n%s", diff)
	}

	gotParams = operation.ParametersIn("body")
	wantParams = []broom.Parameter{}
	if diff := cmp.Diff(wantParams, gotParams); diff != "" {
		t.Errorf("body parameter mismatch (-want +got):\n%s", diff)
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
		Parameters: []broom.Parameter{
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
		Parameters: []broom.Parameter{
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
		Parameters: []broom.Parameter{
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
}
