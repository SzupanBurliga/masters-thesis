package graphql

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"thesis/app/internal/domain"
)

// Limits: odrzuca dokumenty o zbyt dużej głębokości, kosztzie lub liczbie aliasów
// ZANIM uruchomią się resolvery (analiza AST).
type Limits struct {
	MaxDepth   int
	MaxCost    int
	MaxAliases int
}

var _ graphql.HandlerExtension = Limits{}
var _ graphql.OperationContextMutator = Limits{}

func (Limits) ExtensionName() string                   { return "Limits" }
func (Limits) Validate(graphql.ExecutableSchema) error { return nil }

func (l Limits) MutateOperationContext(ctx context.Context, oc *graphql.OperationContext) *gqlerror.Error {
	if oc.Operation == nil {
		return nil
	}
	aliases := 0
	depth := l.depth(oc.Operation.SelectionSet, oc.Doc.Fragments, 0, &aliases)
	if depth > l.MaxDepth {
		return rejected("query depth %d exceeds limit %d", depth, l.MaxDepth)
	}
	if aliases > l.MaxAliases {
		return rejected("alias count %d exceeds limit %d", aliases, l.MaxAliases)
	}
	if cost := l.cost(oc.Operation.SelectionSet, oc.Doc.Fragments, oc.Variables, 0); cost > l.MaxCost {
		return rejected("query cost %d exceeds limit %d", cost, l.MaxCost)
	}
	return nil
}

// rejected zwraca błąd klasy protokołowej, więc serwer odpowie kodem 422 zamiast 200.
func rejected(format string, args ...any) *gqlerror.Error {
	err := gqlerror.Errorf(format, args...)
	errcode.Set(err, errcode.ValidationFailed)
	return err
}

const hardCap = 1 << 30

func (l Limits) depth(set ast.SelectionSet, frags ast.FragmentDefinitionList, d int, aliases *int) int {
	if d > 64 { // zabezpieczenie przed cyklami fragmentów
		return d
	}
	max := d
	for _, s := range set {
		switch f := s.(type) {
		case *ast.Field:
			if f.Alias != "" && f.Alias != f.Name {
				*aliases++
			}
			if len(f.SelectionSet) == 0 {
				continue
			}
			if x := l.depth(f.SelectionSet, frags, d+1, aliases); x > max {
				max = x
			}
		case *ast.InlineFragment:
			if x := l.depth(f.SelectionSet, frags, d, aliases); x > max {
				max = x
			}
		case *ast.FragmentSpread:
			if def := frags.ForName(f.Name); def != nil {
				if x := l.depth(def.SelectionSet, frags, d, aliases); x > max {
					max = x
				}
			}
		}
	}
	return max
}

// cost: skalar = 0, obiekt = 1, lista z limit n = n * koszt elementu (domyślne limit z argumentów).
func (l Limits) cost(set ast.SelectionSet, frags ast.FragmentDefinitionList, vars map[string]any, depth int) int {
	if depth > 64 {
		return hardCap
	}
	total := 0
	for _, s := range set {
		switch f := s.(type) {
		case *ast.Field:
			if len(f.SelectionSet) == 0 {
				continue
			}
			inner := 1 + l.cost(f.SelectionSet, frags, vars, depth+1)
			if f.Definition != nil && f.Definition.Type.Elem != nil {
				inner *= listSize(f, vars)
			}
			total += inner
		case *ast.InlineFragment:
			total += l.cost(f.SelectionSet, frags, vars, depth)
		case *ast.FragmentSpread:
			if def := frags.ForName(f.Name); def != nil {
				total += l.cost(def.SelectionSet, frags, vars, depth)
			}
		}
		if total > hardCap {
			return hardCap
		}
	}
	return total
}

// listSize odwzorowuje rzeczywisty rozmiar listy po stronie serwera: limit < 1 oznacza
// wartość domyślną (clampLimit w store), a limit > domain.MaxLimit jest ucinany.
func listSize(f *ast.Field, vars map[string]any) int {
	def := domain.DefaultListLimit
	if d := f.Definition.Arguments.ForName("limit"); d != nil && d.DefaultValue != nil {
		if v, err := d.DefaultValue.Value(nil); err == nil {
			if n, ok := v.(int64); ok {
				def = int(n)
			}
		}
	} else if f.Arguments.ForName("limit") == nil {
		return 1 // lista bez argumentu limit
	}
	n := def
	if a := f.Arguments.ForName("limit"); a != nil {
		if v, err := a.Value.Value(vars); err == nil {
			switch x := v.(type) {
			case int64:
				n = int(x)
			case int:
				n = x
			case float64:
				n = int(x)
			}
		}
	}
	if n < 1 {
		n = def
	}
	if n > domain.MaxLimit {
		n = domain.MaxLimit
	}
	return n
}

var _ = fmt.Sprintf
