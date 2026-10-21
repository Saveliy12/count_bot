package model

import (
	"slices"
	"strings"
)

type Category string

const (
	CategoryFood          Category = "food"
	CategoryTransport     Category = "transport"
	CategoryHousing       Category = "housing"
	CategoryEntertainment Category = "entertainment"
	CategoryHealth        Category = "health"
	CategoryShopping      Category = "shopping"
	CategoryEducation     Category = "education"
	CategoryTravel        Category = "travel"
	CategorySubscriptions Category = "subscriptions"
	CategoryGifts         Category = "gifts"
	CategoryPets          Category = "pets"
	CategoryPersonalCare  Category = "personal_care"
	CategorySport         Category = "sport"
	CategoryOther         Category = "other"
)

var allCategories = []Category{
	CategoryFood,
	CategoryTransport,
	CategoryHousing,
	CategoryEntertainment,
	CategoryHealth,
	CategoryShopping,
	CategoryEducation,
	CategoryTravel,
	CategorySubscriptions,
	CategoryGifts,
	CategoryPets,
	CategoryPersonalCare,
	CategorySport,
	CategoryOther,
}

var categoryLabels = map[Category]string{
	CategoryFood:          "еда",
	CategoryTransport:     "транспорт",
	CategoryHousing:       "жильё",
	CategoryEntertainment: "развлечения",
	CategoryHealth:        "здоровье",
	CategoryShopping:      "покупки",
	CategoryEducation:     "образование",
	CategoryTravel:        "путешествия",
	CategorySubscriptions: "подписки",
	CategoryGifts:         "подарки",
	CategoryPets:          "питомцы",
	CategoryPersonalCare:  "уход за собой",
	CategorySport:         "спорт",
	CategoryOther:         "другое",
}

// Categories returns all supported categories in a stable order.
func Categories() []Category {
	return slices.Clone(allCategories)
}

// Label returns the Russian display name of the category.
func (c Category) Label() string {
	if label, ok := categoryLabels[c]; ok {
		return label
	}

	return string(c)
}

func (c Category) Valid() bool {
	switch c {
	case CategoryFood,
		CategoryTransport,
		CategoryHousing,
		CategoryEntertainment,
		CategoryHealth,
		CategoryShopping,
		CategoryEducation,
		CategoryTravel,
		CategorySubscriptions,
		CategoryGifts,
		CategoryPets,
		CategoryPersonalCare,
		CategorySport,
		CategoryOther:
		return true
	default:
		return false
	}
}

func ParseCategory(value string) (Category, error) {
	category := Category(strings.ToLower(strings.TrimSpace(value)))
	if !category.Valid() {
		return "", NewValidationError("category", "unsupported category")
	}
	return category, nil
}
