package psql

import (
	"io"

	"github.com/jinzhu/copier"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Select(mods ...mods.QueryMod[*SelectQuery]) *SelectQuery {
	s := &SelectQuery{}
	for _, mod := range mods {
		mod.Apply(s)
	}

	return s
}

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-select.html
type SelectQuery struct {
	expr.With
	expr.Select
	expr.FromItems
	expr.Where
	expr.GroupBy
	expr.Having
	expr.Windows
	expr.Combine
	expr.OrderBy
	expr.Limit
	expr.Offset
	expr.Fetch
	expr.For
}

func (s *SelectQuery) Clone() *SelectQuery {
	var s2 = new(SelectQuery)
	copier.CopyWithOption(s2, s, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
	})

	return s2
}

func (s *SelectQuery) Apply(mods ...mods.QueryMod[*SelectQuery]) {
	for _, mod := range mods {
		mod.Apply(s)
	}
}

func (s SelectQuery) WriteQuery(w io.Writer, start int) ([]any, error) {
	return s.WriteSQL(w, dialect, start)
}

func (s SelectQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, d, start+len(args), s.With,
		len(s.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	selArgs, err := query.ExpressIf(w, d, start+len(args), s.Select, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, selArgs...)

	fromArgs, err := query.ExpressSlice(w, d, start+len(args), s.FromItems.Items, "\nFROM ", ",\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fromArgs...)

	whereArgs, err := query.ExpressIf(w, d, start+len(args), s.Where,
		len(s.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	groupByArgs, err := query.ExpressIf(w, d, start+len(args), s.GroupBy,
		len(s.GroupBy.Groups) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, groupByArgs...)

	havingArgs, err := query.ExpressIf(w, d, start+len(args), s.Having,
		len(s.Having.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, havingArgs...)

	windowArgs, err := query.ExpressIf(w, d, start+len(args), s.Windows,
		len(s.Windows.Windows) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, windowArgs...)

	combineArgs, err := query.ExpressIf(w, d, start+len(args), s.Combine,
		s.Combine.Query != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, combineArgs...)

	orderArgs, err := query.ExpressIf(w, d, start+len(args), s.OrderBy,
		len(s.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	_, err = query.ExpressIf(w, d, start+len(args), s.Limit,
		s.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	_, err = query.ExpressIf(w, d, start+len(args), s.Offset,
		s.Offset.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	_, err = query.ExpressIf(w, d, start+len(args), s.Fetch,
		s.Fetch.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	forArgs, err := query.ExpressIf(w, d, start+len(args), s.For,
		s.For.Strength != "", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, forArgs...)

	w.Write([]byte("\n"))
	return args, nil
}

type SelectQM struct {
	withMod[*SelectQuery]              // For CTEs
	mods.FromMod[*SelectQuery]         // select *FROM*
	joinMod[*expr.FromItem]            // joins, which are mods of the FROM
	mods.TableAliasMod[*expr.FromItem] // Adding an alias to from item
	fromItemMod                        // Dialect specific fromItem mods
}

func (SelectQM) Distinct(expressions ...any) mods.QueryMod[*SelectQuery] {
	return mods.Distinct[*SelectQuery]{
		Distinct: true,
		On:       expressions,
	}
}

func (SelectQM) Select(expressions ...any) mods.QueryMod[*SelectQuery] {
	return mods.Select[*SelectQuery](expressions)
}

func (SelectQM) Where(e query.Expression) mods.QueryMod[*SelectQuery] {
	return mods.Where[*SelectQuery]{e}
}

func (SelectQM) WhereClause(clause string, args ...any) mods.QueryMod[*SelectQuery] {
	return mods.Where[*SelectQuery]{expr.Statement(clause, args...)}
}

func (SelectQM) Having(e query.Expression) mods.QueryMod[*SelectQuery] {
	return mods.Having[*SelectQuery]{e}
}

func (SelectQM) HavingClause(clause string, args ...any) mods.QueryMod[*SelectQuery] {
	return mods.Having[*SelectQuery]{expr.Statement(clause, args...)}
}

func (SelectQM) GroupBy(e any) mods.QueryMod[*SelectQuery] {
	return mods.GroupBy[*SelectQuery]{
		E: e,
	}
}

func (SelectQM) GroupByDistinct(distinct bool) mods.QueryMod[*SelectQuery] {
	return mods.GroupByDistinct[*SelectQuery](distinct)
}

func (SelectQM) Window(name string, definition any) mods.QueryMod[*SelectQuery] {
	return mods.Window[*SelectQuery]{
		Name:      name,
		Definiton: definition,
	}
}

func (SelectQM) OrderBy(e any) mods.QueryMod[*SelectQuery] {
	return orderBy[*SelectQuery](func() expr.OrderDef {
		return expr.OrderDef{
			Expression: e,
		}
	})
}

func (SelectQM) Limit(count int64) mods.QueryMod[*SelectQuery] {
	return mods.Limit[*SelectQuery]{
		Count: count,
	}
}

func (SelectQM) Offset(count int64) mods.QueryMod[*SelectQuery] {
	return mods.Offset[*SelectQuery]{
		Count: count,
	}
}

func (SelectQM) Fetch(count int64, withTies bool) mods.QueryMod[*SelectQuery] {
	return mods.Fetch[*SelectQuery]{
		Count:    &count,
		WithTies: withTies,
	}
}

func (SelectQM) Union(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Union,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) UnionAll(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Union,
		Query:    q,
		All:      true,
	}
}

func (SelectQM) Intersect(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Intersect,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) IntersectAll(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Intersect,
		Query:    q,
		All:      true,
	}
}

func (SelectQM) Except(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Except,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) ExceptAll(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Except,
		Query:    q,
		All:      true,
	}
}

func (SelectQM) ForUpdate(tables ...string) lockChain[*SelectQuery] {
	return lockChain[*SelectQuery](func() expr.For {
		return expr.For{
			Strength: expr.LockStrengthUpdate,
			Tables:   tables,
		}
	})
}

func (SelectQM) ForNoKeyUpdate(tables ...string) lockChain[*SelectQuery] {
	return lockChain[*SelectQuery](func() expr.For {
		return expr.For{
			Strength: expr.LockStrengthNoKeyUpdate,
			Tables:   tables,
		}
	})
}

func (SelectQM) ForShare(tables ...string) lockChain[*SelectQuery] {
	return lockChain[*SelectQuery](func() expr.For {
		return expr.For{
			Strength: expr.LockStrengthShare,
			Tables:   tables,
		}
	})
}

func (SelectQM) ForKeyShare(tables ...string) lockChain[*SelectQuery] {
	return lockChain[*SelectQuery](func() expr.For {
		return expr.For{
			Strength: expr.LockStrengthKeyShare,
			Tables:   tables,
		}
	})
}
