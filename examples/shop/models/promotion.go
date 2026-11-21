package models

import "github.com/epistemer/i18n"

// PromotionType is the promotion_type enum.
type PromotionType string

const (
	PromotionTypeFixed      PromotionType = "fixed"
	PromotionTypeAbsolute   PromotionType = "absolute"
	PromotionTypePercentage PromotionType = "percentage"
	PromotionTypePoints     PromotionType = "points"
)

func AllPromotionTypeValues() []PromotionType {
	return []PromotionType{PromotionTypeFixed, PromotionTypeAbsolute, PromotionTypePercentage, PromotionTypePoints}
}

func (t PromotionType) Label() i18n.TKer { return labels["enum.promotion_type."+string(t)] }
func (t PromotionType) Value() string    { return string(t) }

// DiscountBase is the discount_base enum.
type DiscountBase string

const (
	DiscountBaseSubtotal DiscountBase = "subtotal"
	DiscountBaseTotal    DiscountBase = "total"
)

func AllDiscountBaseValues() []DiscountBase {
	return []DiscountBase{DiscountBaseSubtotal, DiscountBaseTotal}
}

func (t DiscountBase) Label() i18n.TKer { return labels["enum.discount_base."+string(t)] }
func (t DiscountBase) Value() string    { return string(t) }

// ConstraintType is the constraint_type enum.
type ConstraintType string

const (
	ConstraintTypeMonetary ConstraintType = "monetary"
	ConstraintTypeQuantity ConstraintType = "quantity"
)

func AllConstraintTypeValues() []ConstraintType {
	return []ConstraintType{ConstraintTypeMonetary, ConstraintTypeQuantity}
}

func (t ConstraintType) Label() i18n.TKer { return labels["enum.constraint_type."+string(t)] }
func (t ConstraintType) Value() string    { return string(t) }

var labels = func() map[string]i18n.TKer {
	m := map[string]i18n.TKer{}
	for _, v := range AllPromotionTypeValues() {
		m["enum.promotion_type."+string(v)] = i18n.Key("enum.promotion_type."+string(v), string(v))
	}
	for _, v := range AllDiscountBaseValues() {
		m["enum.discount_base."+string(v)] = i18n.Key("enum.discount_base."+string(v), string(v))
	}
	for _, v := range AllConstraintTypeValues() {
		m["enum.constraint_type."+string(v)] = i18n.Key("enum.constraint_type."+string(v), string(v))
	}
	return m
}()
