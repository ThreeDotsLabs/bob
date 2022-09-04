package drivers

import (
	"github.com/stephenafamo/bob/orm"
)

func BuildRelationships(tables []Table) map[string][]orm.Relationship {
	relationships := map[string][]orm.Relationship{}

	tableNameMap := make(map[string]Table, len(tables))
	for _, t := range tables {
		tableNameMap[t.Name] = t
	}

	for _, t1 := range tables {
		fkUniqueMap := make(map[string][2]bool, len(t1.FKeys))

		// Build BelongsTo, ToOne and ToMany
		for _, fk := range t1.FKeys {
			localUnique := hasExactUnique(t1, fk.Columns...)

			t2, ok := tableNameMap[fk.ForeignTable]
			if !ok {
				continue // no matching target table
			}

			foreignUnique := hasExactUnique(t2, fk.ForeignColumns...)
			fkUniqueMap[fk.Name] = [2]bool{localUnique, foreignUnique}

			pair1 := make(map[string]string, len(fk.Columns))
			pair2 := make(map[string]string, len(fk.Columns))
			for index, localCol := range fk.Columns {
				foreignCol := fk.ForeignColumns[index]
				pair1[localCol] = foreignCol
				pair2[foreignCol] = localCol
			}

			relationships[t1.Name] = append(relationships[t1.Name], orm.Relationship{
				Name: fk.Name,
				Sides: []orm.RelSide{{
					From:     t1.Name,
					To:       t2.Name,
					Pairs:    pair1,
					ToUnique: foreignUnique,
				}},
			})

			if !t1.IsJoinTable && t1.Name != t2.Name {
				relationships[t2.Name] = append(relationships[t2.Name], orm.Relationship{
					Name: fk.Name,
					Sides: []orm.RelSide{{
						From:     t2.Name,
						To:       t1.Name,
						Pairs:    pair2,
						ToUnique: localUnique,
					}},
				})
			}
		}

		if !t1.IsJoinTable {
			continue
		}

		// Build ManyToMany
		rels := relationships[t1.Name]
		if len(rels) != 2 {
			panic("join table does not have 2 relationships")
		}
		r1, r2 := rels[0], rels[1]

		relationships[r1.Sides[0].To] = append(relationships[r1.Sides[0].To], orm.Relationship{
			Name: r2.Name,
			Sides: []orm.RelSide{
				{
					From:     r1.Sides[0].To,
					To:       t1.Name,
					Pairs:    invertMap(r1.Sides[0].Pairs),
					ToUnique: fkUniqueMap[r1.Name][0],
				},
				{
					From:     t1.Name,
					To:       r2.Sides[0].To,
					Pairs:    r2.Sides[0].Pairs,
					ToUnique: fkUniqueMap[r1.Name][1],
				},
			},
		})
		relationships[r2.Sides[0].To] = append(relationships[r2.Sides[0].To], orm.Relationship{
			Name: r1.Name,
			Sides: []orm.RelSide{
				{
					From:     r2.Sides[0].To,
					To:       t1.Name,
					Pairs:    invertMap(r2.Sides[0].Pairs),
					ToUnique: fkUniqueMap[r2.Name][0],
				},
				{
					From:     t1.Name,
					To:       r1.Sides[0].To,
					Pairs:    r1.Sides[0].Pairs,
					ToUnique: fkUniqueMap[r2.Name][1],
				},
			},
		})
	}

	return relationships
}

// Returns true if the table has a unique constraint on exactly these columns
func hasExactUnique(t Table, cols ...string) bool {
	if len(cols) == 0 {
		return false
	}

	// Primary keys are unique
	if t.PKey != nil && sliceMatch(t.PKey.Columns, cols) {
		return true
	}

	// Check other unique constrints
	for _, u := range t.Uniques {
		if sliceMatch(u.Columns, cols) {
			return true
		}
	}

	return false
}

func sliceMatch[T comparable, Ts ~[]T](a, b Ts) bool {
	if len(a) != len(b) {
		return false
	}

	if len(a) == 0 {
		return false
	}

	var matches int
	for _, v1 := range a {
		for _, v2 := range b {
			if v1 == v2 {
				matches++
			}
		}
	}

	return matches == len(a)
}

func invertMap[T comparable, Tm ~map[T]T](from Tm) Tm {
	to := make(Tm, len(from))
	for k, v := range from {
		to[v] = k
	}

	return to
}

// Has no matching elements
func distinctElems[T comparable, Ts ~[]T](a, b Ts) bool {
	for _, v1 := range a {
		for _, v2 := range b {
			if v1 == v2 {
				return false
			}
		}
	}

	return true
}

// A composite primary key involving two columns
// Both primary key columns are also foreign keys
func IsJoinTable(t Table) bool {
	if t.PKey == nil {
		return false
	}

	// Must have exactly 2 foreign keys
	if len(t.FKeys) != 2 {
		return false
	}

	// Number of columns must be the number of primary key columns
	if len(t.Columns) != len(t.PKey.Columns) {
		return false
	}

	// length of both foreign keys must be the total length of the columns
	if len(t.Columns) != (len(t.FKeys[0].Columns) + len(t.FKeys[1].Columns)) {
		return false
	}

	// both foreign keys must have distinct columns
	if !distinctElems(t.FKeys[0].Columns, t.FKeys[1].Columns) {
		return false
	}

	// It is a join table!!!
	return true
}
