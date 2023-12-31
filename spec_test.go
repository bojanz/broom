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
				Header: broom.ParameterList{vendorParam},
				Query: broom.ParameterList{
					broom.Parameter{
						In:          "query",
						Name:        "filter[owner_id]",
						Description: "Allows filtering by owner_id.",
						Type:        "string",
						Deprecated:  true,
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
		},
		broom.Operation{
			ID:          "create-product",
			Summary:     "Create product",
			Description: "Creates a new product.",
			Tag:         "Products",
			Method:      "POST",
			Path:        "/products",
			Parameters: broom.Parameters{
				Header: broom.ParameterList{vendorParam},
				Body: broom.ParameterList{
					broom.Parameter{
						In:          "body",
						Name:        "owner_id",
						Description: "ID of the owner. Defaults to the requester.",
						Type:        "string",
					},
					broom.Parameter{
						In:          "body",
						Name:        "name",
						Description: "The product name.",
						Type:        "string",
						Required:    true,
					},
					broom.Parameter{
						In:          "body",
						Name:        "sku",
						Description: "The product sku.",
						Type:        "string",
					},
					broom.Parameter{
						In:          "body",
						Name:        "description",
						Description: "The product description.",
						Type:        "string",
					},
					broom.Parameter{
						In:          "body",
						Name:        "price",
						Description: "The product price, in cents.",
						Type:        "integer",
						Example:     "1099",
						Required:    true,
					},
					broom.Parameter{
						In:          "body",
						Name:        "currency_code",
						Description: "The currency code.",
						Type:        "string",
						Enum:        []string{"EUR", "USD"},
						Default:     "USD",
						Required:    true,
					},
					broom.Parameter{
						In:          "body",
						Name:        "status",
						Description: "Whether the product is available for purchase.",
						Type:        "boolean",
						Default:     "true",
					},
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
			Parameters: broom.Parameters{
				Header: broom.ParameterList{vendorParam},
				Path:   broom.ParameterList{idParam},
			},
		},
		broom.Operation{
			ID:          "update-product",
			Summary:     "Update product",
			Description: "Updates the specified product.",
			Tag:         "Products",
			Method:      "PATCH",
			Path:        "/products/{product_id}",
			Parameters: broom.Parameters{
				Header: broom.ParameterList{vendorParam},
				Path:   broom.ParameterList{idParam},
				Body: broom.ParameterList{
					broom.Parameter{
						In:          "body",
						Name:        "name",
						Description: "The product name.",
						Type:        "string",
					},
					broom.Parameter{
						In:          "body",
						Name:        "sku",
						Description: "The product sku.",
						Type:        "string",
						Deprecated:  true,
					},
					broom.Parameter{
						In:          "body",
						Name:        "description",
						Description: "The product description.",
						Type:        "string",
					},
					broom.Parameter{
						In:          "body",
						Name:        "price",
						Description: "The product price, in cents.",
						Type:        "integer",
					},
					broom.Parameter{
						In:          "body",
						Name:        "currency_code",
						Description: "The currency code.",
						Type:        "string",
						Enum:        []string{"EUR", "USD"},
					},
					broom.Parameter{
						In:          "body",
						Name:        "status",
						Description: "Whether the product is available for purchase.",
						Type:        "boolean",
					},
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
			Parameters: broom.Parameters{
				Header: broom.ParameterList{vendorParam},
				Path:   broom.ParameterList{idParam},
			},
		},
	}

	gotOps, err := broom.LoadOperations("testdata/openapi3.yaml")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if diff := cmp.Diff(wantOps, gotOps); diff != "" {
		t.Errorf("operation mismatch (-want +got):\n%s", diff)
	}
}
