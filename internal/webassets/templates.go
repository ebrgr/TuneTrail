package webassets

import (
	"fmt"
	"html/template"
	"text/template/parse"
)

// ValidateTemplates rejects undefined or bodyless partials and identifies the referring template.
func ValidateTemplates(templates *template.Template) error {
	var visit func(parse.Node) error
	visit = func(node parse.Node) error {
		switch n := node.(type) {
		case *parse.ListNode:
			if n == nil {
				return nil
			}
			for _, child := range n.Nodes {
				if err := visit(child); err != nil {
					return err
				}
			}
		case *parse.TemplateNode:
			if target := templates.Lookup(n.Name); target == nil || target.Tree == nil {
				return fmt.Errorf("undefined or bodyless template %q", n.Name)
			}
		case *parse.IfNode:
			if err := visit(n.List); err != nil {
				return err
			}
			return visit(n.ElseList)
		case *parse.RangeNode:
			if err := visit(n.List); err != nil {
				return err
			}
			return visit(n.ElseList)
		case *parse.WithNode:
			if err := visit(n.List); err != nil {
				return err
			}
			return visit(n.ElseList)
		}
		return nil
	}
	for _, t := range templates.Templates() {
		if t.Tree != nil {
			if err := visit(t.Tree.Root); err != nil {
				return fmt.Errorf("template %q: %w", t.Name(), err)
			}
		}
	}
	return nil
}
