package rqlParser

import (
	"fmt"
	"strconv"
	"strings"
)

type TranslatorOpFunc func(*RqlNode) (string, error)

type SqlTranslator struct {
	rootNode  *RqlRootNode
	sqlOpsDic map[string]TranslatorOpFunc
}

func (st *SqlTranslator) SetOpFunc(op string, f TranslatorOpFunc) {
	st.sqlOpsDic[strings.ToUpper(op)] = f
}

func (st *SqlTranslator) DeleteOpFunc(op string) {
	delete(st.sqlOpsDic, strings.ToUpper(op))
}

func (st *SqlTranslator) GetSimpleTranslatorFunc(op string, betweenParenthesis bool, quoteStr bool) TranslatorOpFunc {
	return TranslatorOpFunc(func(n *RqlNode) (s string, err error) {
		sep := ""
		quote := ""

		for _, a := range n.Args {
			s = s + sep
			switch a.(type) {
			case string:
				v := a.(string)
				if _, err := strconv.Atoi(v); err != nil {
					v = quote + v + quote
				}
				s = s + v
			case *RqlNode:
				var _s string
				_s, err = st.Where(a.(*RqlNode))
				if err != nil {
					return "", err
				}
				s = s + _s
			}

			sep = " " + op + " "
			if quoteStr {
				quote = "'"
			}
		}

		if betweenParenthesis {
			s = "(" + s + ")"
		}
		return
	})
}

func (st *SqlTranslator) Where(n *RqlNode) (string, error) {
	f := st.sqlOpsDic[strings.ToUpper(n.Op)]
	if f == nil {
		return "", fmt.Errorf("No TranslatorOpFunc for op : '%s'", n.Op)
	}
	return f(n)
}

func (st *SqlTranslator) Limit() (sql string) {
	limit := st.rootNode.Limit()
	if limit != "" && strings.ToUpper(limit) != "INFINITY" {
		sql = " LIMIT " + limit
	}
	return
}

func (st *SqlTranslator) Offset() (sql string) {
	if st.rootNode.Offset() != "" {
		sql = " OFFSET " + st.rootNode.Offset()
	}
	return
}

func (st *SqlTranslator) Sort() (sql string) {
	sorts := st.rootNode.Sort()
	if len(sorts) > 0 {
		sql = " SORT BY "
		sep := ""
		for _, sort := range sorts {
			sql = sql + sep + sort.by
			if sort.desc {
				sql = sql + " DESC"
			}
			sep = ", "
		}
	}

	return
}

func (st *SqlTranslator) Sql() (string, error) {
	sql, err := st.Where(st.rootNode.Node)
	if err != nil {
		return sql, err
	}
	sql = sql + st.Sort() + st.Limit() + st.Offset()

	return sql, nil
}

func NewSqlTranslator(r *RqlRootNode) (st *SqlTranslator) {
	st = &SqlTranslator{r, map[string]TranslatorOpFunc{}}

	st.SetOpFunc("AND", st.GetSimpleTranslatorFunc("AND", true, false))
	st.SetOpFunc("OR", st.GetSimpleTranslatorFunc("OR", true, false))
	st.SetOpFunc("EQ", st.GetSimpleTranslatorFunc("=", false, true))
	st.SetOpFunc("LIKE", st.GetSimpleTranslatorFunc("LIKE", false, true))
	st.SetOpFunc("MATCH", st.GetSimpleTranslatorFunc("ILIKE", false, true))

	return
}
