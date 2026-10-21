package bot

import "html"

func escapeHTML(value string) string {
	return html.EscapeString(value)
}

func bold(value string) string {
	return "<b>" + escapeHTML(value) + "</b>"
}

func code(value string) string {
	return "<code>" + escapeHTML(value) + "</code>"
}
