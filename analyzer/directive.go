package analyzer

import (
	"go/ast"
	"strconv"
	"strings"
)

// directive holds the per-type configuration parsed from a comment attached to
// a type declaration. It is the inline counterpart to a Config override: it
// lives next to the code it excuses, so the exception travels with the type and
// disappears when the type is split up — unlike a central exclude-list, which
// rots.
//
// Supported forms, on the line(s) immediately above (or trailing) the type:
//
//	//plimsoll:ignore                       — skip this type entirely
//	//plimsoll:max-methods=60               — override the total-method cap
//	//plimsoll:max-exported-methods=25      — override the exported-method cap
//	//plimsoll:max-fields=30                — override the exported-field cap
//
// And, on the package doc comment, for the package scope:
//
//	//plimsoll:ignore                       — skip this package entirely
//	//plimsoll:max-exported-types=80        — override the exported-type cap
//
// Multiple settings may share one comment group, one per line. A directive
// ALWAYS wins over file/default config — it is the most local, most visible
// statement of intent.
type directive struct {
	ignore        bool
	maxMethods    *int
	maxExpMethods *int
	maxFields     *int
	maxExpTypes   *int
	hasOverride   bool // any of the above was set
}

const directivePrefix = "//plimsoll:"

// parseDirective scans a comment group for plimsoll directives. A nil group
// yields the zero directive (no settings).
func parseDirective(cg *ast.CommentGroup) directive {
	var d directive
	if cg == nil {
		return d
	}
	for _, c := range cg.List {
		text := strings.TrimSpace(c.Text)
		if !strings.HasPrefix(text, directivePrefix) {
			continue
		}
		body := strings.TrimSpace(strings.TrimPrefix(text, directivePrefix))
		key, val, hasVal := cut(body, "=")
		switch key {
		case "ignore":
			d.ignore = true
			d.hasOverride = true
		case "max-methods":
			if hasVal {
				if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					d.maxMethods = &n
					d.hasOverride = true
				}
			}
		case "max-exported-methods":
			if hasVal {
				if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					d.maxExpMethods = &n
					d.hasOverride = true
				}
			}
		case "max-fields":
			if hasVal {
				if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					d.maxFields = &n
					d.hasOverride = true
				}
			}
		case "max-exported-types":
			if hasVal {
				if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
					d.maxExpTypes = &n
					d.hasOverride = true
				}
			}
		}
	}
	return d
}

func cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
