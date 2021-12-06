// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom_test

import (
	"testing"

	"github.com/bojanz/broom"
	"github.com/google/go-cmp/cmp"
)

func TestLoadOperations(t *testing.T) {
	idParam := broom.Parameter{
		In:          "path",
		Name:        "product_id",
		Description: "The ID of the product.",
		Type:        "string",
		Default:     nil,
		Required:    true,
	}
	vendorParam := broom.Parameter{
		In:          "header",
		Name:        "X-Vendor",
		Description: "The vendor.",
		Type:        "string",
	}
	wantOps := broom.Operations{
		broom.Operation{
			ID:          "list-products",
			Summary:     "List products",
			Description: "Retrieves a list of products matching the specified criteria.",
			Tag:         "Products",
			Method:      "GET",
			Path:        "/products",
			Parameters: broom.Parameters{
				vendorParam,
				broom.Parameter{
					In:          "query",
					Name:        "filter[owner_id]",
					Description: "Allows filtering by owner_id.",
					Type:        "string",
				},
				broom.Parameter{
					In:          "query",
					Name:        "page[before]",
					Description: "Shows 50 products before the given ID.",
					Type:        "string",
				},
				broom.Parameter{
					In:          "query",
					Name:        "page[after]",
					Description: "Shows 50 products after the given ID.",
					Type:        "string",
				},
				broom.Parameter{
					In:          "query",
					Name:        "sort",
					Description: "Allows sorting by a single field.\nUse a dash (\"-\") to sort descending.",
					Type:        "string",
					Default:     "created_at",
				},
			},
		},
		broom.Operation{
			ID:          "create-product",
			Summary:     "Create product",
			Description: "Creates a new product.",
			Tag:         "Products",
			Method:      "POST",
			Path:        "/products",
			Parameters: broom.Parameters{
				vendorParam,
				broom.Parameter{
					In:          "body",
					Name:        "currency_code",
					Description: "The currency code.",
					Type:        "string",
					Enum:        []string{"EUR", "USD"},
					Default:     interface{}("USD"),
					Required:    true,
				},
				broom.Parameter{
					In:          "body",
					Name:        "name",
					Description: "The product name.",
					Type:        "string",
					Default:     nil,
					Required:    true,
				},
				broom.Parameter{
					In:          "body",
					Name:        "owner_id",
					Description: "ID of the owner. Defaults to the requester.",
					Type:        "string",
				},
				broom.Parameter{
					In:          "body",
					Name:        "price",
					Description: "The product price, in cents.",
					Type:        "integer",
					Default:     interface{}(float64(1099)),
					Required:    true,
				},
				broom.Parameter{
					In:          "body",
					Name:        "sku",
					Description: "The product sku.",
					Type:        "string",
					Default:     nil,
				},
				broom.Parameter{
					In:          "body",
					Name:        "status",
					Description: "Whether the product is available for purchase.",
					Type:        "boolean",
					Default:     interface{}(true),
				},
			},
			BodyFormat: "application/json",
		},
		broom.Operation{
			ID:          "get-product",
			Summary:     "Get product",
			Description: "Retrieves the specified product.",
			Tag:         "Products",
			Method:      "GET",
			Path:        "/products/{product_id}",
			Parameters:  broom.Parameters{idParam, vendorParam},
		},
		broom.Operation{
			ID:          "update-product",
			Summary:     "Update product",
			Description: "Updates the specified product.",
			Tag:         "Products",
			Method:      "PATCH",
			Path:        "/products/{product_id}",
			Parameters: broom.Parameters{
				idParam,
				vendorParam,
				broom.Parameter{
					In:          "body",
					Name:        "currency_code",
					Description: "The currency code.",
					Type:        "string",
					Enum:        []string{"EUR", "USD"},
					Default:     nil,
				},
				broom.Parameter{
					In:          "body",
					Name:        "name",
					Description: "The product name.",
					Type:        "string",
					Default:     nil,
				},
				broom.Parameter{
					In:          "body",
					Name:        "price",
					Description: "The product price, in cents.",
					Type:        "integer",
					Default:     nil,
				},
				broom.Parameter{
					In:          "body",
					Name:        "sku",
					Description: "The product sku.",
					Type:        "string",
				},
				broom.Parameter{
					In:          "body",
					Name:        "status",
					Description: "Whether the product is available for purchase.",
					Type:        "boolean",
					Default:     nil,
				},
			},
			BodyFormat: "application/json",
		},
		broom.Operation{
			ID:          "delete-product",
			Summary:     "Delete product",
			Description: "Deletes the specified product.",
			Tag:         "Products",
			Method:      "DELETE",
			Path:        "/products/{product_id}",
			Parameters:  broom.Parameters{idParam, vendorParam},
		},
	}

	gotOps, err := broom.LoadOperations("testdata/openapi3.yaml")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if diff := cmp.Diff(wantOps, gotOps); diff != "" {
		t.Errorf("operation mismatch (-want +got):\n%s", diff)
	}

	gotOps, err = broom.LoadOperations("testdata/swagger.yaml")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if diff := cmp.Diff(wantOps, gotOps); diff != "" {
		t.Errorf("operation mismatch (-want +got):\n%s", diff)
	}
}
