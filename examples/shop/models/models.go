// Package models stands in for what sqlc and github.com/epistemer/models
// generate for the shop's schema: an enum with Label(), Value() and its
// values, and the table labels. Written by hand so the example needs no
// database.
package models

import (
	"github.com/epistemer/i18n"
	"github.com/epistemer/models/option"
)

// StoreKind is the store_kind enum.
type StoreKind string

const (
	StoreKindShop      StoreKind = "shop"
	StoreKindWarehouse StoreKind = "warehouse"
	StoreKindKiosk     StoreKind = "kiosk"
)

// AllStoreKindValues is what sqlc's emit_all_enum_values gives.
func AllStoreKindValues() []StoreKind {
	return []StoreKind{StoreKindShop, StoreKindWarehouse, StoreKindKiosk}
}

var storeKindLabels = map[StoreKind]i18n.TKer{
	StoreKindShop:      i18n.Key("enum.store_kind.shop", "shop"),
	StoreKindWarehouse: i18n.Key("enum.store_kind.warehouse", "warehouse"),
	StoreKindKiosk:     i18n.Key("enum.store_kind.kiosk", "kiosk"),
}

// Label is the value's label.
func (t StoreKind) Label() i18n.TKer { return storeKindLabels[t] }

// Value is the value sent and stored.
func (t StoreKind) Value() string { return string(t) }

// AllStoreKindOptions lists the values as choices.
func AllStoreKindOptions() []option.SelectOption {
	return []option.SelectOption{StoreKindShop, StoreKindWarehouse, StoreKindKiosk}
}

// Labels of the store table.
var (
	L10nStore_Name        = i18n.Key("table.store.Name", "Name")
	L10nStore_Address     = i18n.Key("table.store.Address", "Address")
	L10nStore_Kind        = i18n.Key("table.store.Kind", "Kind")
	L10nStore_Description = i18n.Key("table.store.Description", "Description")
	L10nStore_Logo        = i18n.Key("table.store.Logo", "Logo")
)

// Store is a row of the store table.
type Store struct {
	ID          int64
	Name        string
	Address     string
	Kind        StoreKind
	Description *string
	LogoURL     string
}
