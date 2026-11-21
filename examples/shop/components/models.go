package components

// Order is a toy shop order for examples and tests.
type Order struct {
	ID       string
	Customer string
	Total    string
}

// Product is a toy shop product for examples and tests.
type Product struct {
	ID    string
	Name  string
	Price string
}
