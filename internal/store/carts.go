package store

import "errors"

type AddCartItemInput struct {
	ProductReference string `json:"product_reference"`
	Quantity         int    `json:"quantity"`
}

func (in AddCartItemInput) Validate() error {
	if in.ProductReference == "" {
		return errors.New("ProductReference est essentiel")
	}
	if in.Quantity < 1 {
		return errors.New("Quantity doit être supérieur à 0")
	}
	return nil
}
