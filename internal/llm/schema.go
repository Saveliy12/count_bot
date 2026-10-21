package llm

import "github.com/Saveliy12/expenses/internal/model"

func intentSchema() map[string]any {
	categoryValues := make([]any, 0, len(model.Categories())+1)
	for _, category := range model.Categories() {
		categoryValues = append(categoryValues, string(category))
	}
	categoryValues = append(categoryValues, nil)

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []any{"create", "list", "update", "delete", "clarify"},
			},
			"amount": map[string]any{
				"type": []any{"string", "null"},
			},
			"old_amount": map[string]any{
				"type": []any{"string", "null"},
			},
			"category": map[string]any{
				"type": []any{"string", "null"},
				"enum": categoryValues,
			},
			"date_hint": map[string]any{
				"type": []any{"string", "null"},
			},
			"comment": map[string]any{
				"type": []any{"string", "null"},
			},
			"clear_comment": map[string]any{
				"type": "boolean",
			},
			"target_mode": map[string]any{
				"type": []any{"string", "null"},
				"enum": []any{"latest", "id", "query", nil},
			},
			"target_id": map[string]any{
				"type": []any{"integer", "null"},
			},
			"target_query": map[string]any{
				"type": []any{"string", "null"},
			},
			"target_date_hint": map[string]any{
				"type": []any{"string", "null"},
			},
			"target_category": map[string]any{
				"type": []any{"string", "null"},
				"enum": categoryValues,
			},
			"period_preset": map[string]any{
				"type": []any{"string", "null"},
				"enum": []any{"today", "yesterday", "week", "month", "year", "all", nil},
			},
			"period_from": map[string]any{
				"type": []any{"string", "null"},
			},
			"period_to": map[string]any{
				"type": []any{"string", "null"},
			},
			"clarification": map[string]any{
				"type": []any{"string", "null"},
			},
		},
		"required": []any{
			"action",
			"amount",
			"old_amount",
			"category",
			"date_hint",
			"comment",
			"clear_comment",
			"target_mode",
			"target_id",
			"target_query",
			"target_date_hint",
			"target_category",
			"period_preset",
			"period_from",
			"period_to",
			"clarification",
		},
		"additionalProperties": false,
	}
}
