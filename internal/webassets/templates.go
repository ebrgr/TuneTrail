package webassets

import (
	"fmt"
	"html/template"
	"text/template/parse"
)

// ValidateTemplates rejects missing partials and unknown or dynamic asset calls before startup.
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
		case *parse.ActionNode:
			return visit(n.Pipe)
		case *parse.PipeNode:
			if n == nil {
				return nil
			}
			for _, command := range n.Cmds {
				if err := visit(command); err != nil {
					return err
				}
			}
		case *parse.ChainNode:
			return visit(n.Node)
		case *parse.CommandNode:
			if len(n.Args) > 0 {
				if id, ok := n.Args[0].(*parse.IdentifierNode); ok && id.Ident == "asset" {
					if len(n.Args) != 2 {
						return fmt.Errorf("asset requires exactly one literal name")
					}
					name, ok := n.Args[1].(*parse.StringNode)
					if !ok {
						return fmt.Errorf("asset requires a literal name")
					}
					s, err := Current()
					if err != nil {
						return err
					}
					if _, ok := s.Asset(name.Text); !ok {
						return fmt.Errorf("unknown public asset %q", name.Text)
					}
				}
			}
			for _, arg := range n.Args {
				if err := visit(arg); err != nil {
					return err
				}
			}
		case *parse.TemplateNode:
			if err := visit(n.Pipe); err != nil {
				return err
			}
			if target := templates.Lookup(n.Name); target == nil || target.Tree == nil {
				return fmt.Errorf("undefined or bodyless template %q", n.Name)
			}
		case *parse.IfNode:
			if err := visit(n.Pipe); err != nil {
				return err
			}
			if err := visit(n.List); err != nil {
				return err
			}
			return visit(n.ElseList)
		case *parse.RangeNode:
			if err := visit(n.Pipe); err != nil {
				return err
			}
			if err := visit(n.List); err != nil {
				return err
			}
			return visit(n.ElseList)
		case *parse.WithNode:
			if err := visit(n.Pipe); err != nil {
				return err
			}
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
