package rqlParser

import (
	"fmt"
	"net/url"
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

func (st *SqlTranslator) Where() (string, error) {
	if st.rootNode == nil {
		return "", nil
	}
	return st.where(st.rootNode.Node)
}

func (st *SqlTranslator) where(n *RqlNode) (string, error) {
	if n == nil {
		return ``, nil
	}
	f := st.sqlOpsDic[strings.ToUpper(n.Op)]
	if f == nil {
		return "", fmt.Errorf("No TranslatorOpFunc for op : '%s'", n.Op)
	}
	return f(n)
}

func (st *SqlTranslator) Limit() (sql string) {
	if st.rootNode == nil {
		return
	}
	limit := st.rootNode.Limit()
	if limit != "" && strings.ToUpper(limit) != "INFINITY" {
		sql = " LIMIT " + limit
	}
	return
}

func (st *SqlTranslator) Offset() (sql string) {
	if st.rootNode != nil && st.rootNode.Offset() != "" {
		sql = " OFFSET " + st.rootNode.Offset()
	}
	return
}

func (st *SqlTranslator) Sort() (sql string) {
	if st.rootNode == nil {
		return
	}
	sorts := st.rootNode.Sort()
	if len(sorts) > 0 {
		sql = " ORDER BY "
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

func (st *SqlTranslator) Sql() (sql string, err error) {
	var where string

	where, err = st.Where()
	if err != nil {
		return
	}

	if len(where) > 0 {
		sql = `WHERE ` + where
	}

	sort := st.Sort()
	if len(sort) > 0 {
		sql += `ORDER BY ` + sort
	}

	limit := st.Limit()
	if len(limit) > 0 {
		sql += `LIMIT ` + limit
	}

	offset := st.Offset()
	if len(offset) > 0 {
		sql += `OFFSET ` + offset
	}

	return sql, nil
}

func NewSqlTranslator(r *RqlRootNode) (st *SqlTranslator) {
	st = &SqlTranslator{r, map[string]TranslatorOpFunc{}}

	starToPercentFunc := AlterStringFunc(func(s string) (string, error) {
		v, err := url.QueryUnescape(s)
		if err != nil {
			return ``, err
		}
		return strings.Replace(Quote(v), `*`, `%`, -1), nil
	})

	st.SetOpFunc("AND", st.GetAndOrTranslatorOpFunc("AND"))
	st.SetOpFunc("OR", st.GetAndOrTranslatorOpFunc("OR"))

	st.SetOpFunc("NE", st.GetEqualityTranslatorOpFunc("!=", "IS NOT"))
	st.SetOpFunc("EQ", st.GetEqualityTranslatorOpFunc("=", "IS"))

	st.SetOpFunc("LIKE", st.GetFieldValueTranslatorFunc("LIKE", starToPercentFunc))
	st.SetOpFunc("MATCH", st.GetFieldValueTranslatorFunc("ILIKE", starToPercentFunc))
	st.SetOpFunc("GT", st.GetFieldValueTranslatorFunc(">", nil))
	st.SetOpFunc("LT", st.GetFieldValueTranslatorFunc("<", nil))
	st.SetOpFunc("GE", st.GetFieldValueTranslatorFunc(">=", nil))
	st.SetOpFunc("LE", st.GetFieldValueTranslatorFunc("<=", nil))
	st.SetOpFunc("SUM", st.GetOpFirstTranslatorFunc("SUM", nil))
	st.SetOpFunc("NOT", st.GetOpFirstTranslatorFunc("NOT", nil))

	return
}

func (st *SqlTranslator) GetEqualityTranslatorOpFunc(op, specialOp string) TranslatorOpFunc {
	return TranslatorOpFunc(func(n *RqlNode) (s string, err error) {
		value, err := url.QueryUnescape(n.Args[1].(string))
		if err != nil {
			return "", err
		}

		if value == `null` || value == `true` || value == `false` {
			field := n.Args[0].(string)
			if !IsValidField(field) {
				return ``, fmt.Errorf("Invalid field name : %s", field)
			}

			return fmt.Sprintf("(%s %s %s)", field, specialOp, strings.ToUpper(value)), nil
		}

		return st.GetFieldValueTranslatorFunc(op, nil)(n)
	})
}

func (st *SqlTranslator) GetAndOrTranslatorOpFunc(op string) TranslatorOpFunc {
	return TranslatorOpFunc(func(n *RqlNode) (s string, err error) {
		sep := ""

		for _, a := range n.Args {
			s = s + sep
			switch v := a.(type) {
			case string:
				if !IsValidField(v) {
					return "", fmt.Errorf("Invalid field name : %s", v)
				}
				s = s + v
			case *RqlNode:
				var _s string
				_s, err = st.where(v)
				if err != nil {
					return "", err
				}
				s = s + _s
			}

			sep = " " + op + " "
		}

		return "(" + s + ")", nil
	})
}

type AlterStringFunc func(string) (string, error)

func (st *SqlTranslator) GetFieldValueTranslatorFunc(op string, valueAlterFunc AlterStringFunc) TranslatorOpFunc {
	return TranslatorOpFunc(func(n *RqlNode) (s string, err error) {
		sep := ""

		for i, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case string:
				var _s string
				if i == 0 {
					if IsValidField(v) {
						_s = v
					} else {
						return "", fmt.Errorf("First argument must be a valid field name (arg: %s)", v)
					}
				} else {
					_, err := strconv.ParseInt(v, 10, 64)
					if err == nil {
						_s = v
					} else if valueAlterFunc != nil {
						_s, err = valueAlterFunc(v)
						if err != nil {
							return "", err
						}
					} else {
						_s = Quote(v)
					}
				}

				s += _s
			case *RqlNode:
				var _s string
				_s, err = st.where(v)
				if err != nil {
					return "", err
				}
				s = s + _s
			}

			sep = " " + op + " "
		}

		return "(" + s + ")", nil
	})
}

func (st *SqlTranslator) GetOpFirstTranslatorFunc(op string, valueAlterFunc AlterStringFunc) TranslatorOpFunc {
	return TranslatorOpFunc(func(n *RqlNode) (s string, err error) {
		sep := ""

		for _, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case string:
				var _s string
				_, err := strconv.ParseInt(v, 10, 64)
				if err == nil || IsValidField(v) {
					_s = v
				} else if valueAlterFunc != nil {
					_s, err = valueAlterFunc(v)
					if err != nil {
						return "", err
					}
				} else {
					_s = Quote(v)
				}

				s += _s
			case *RqlNode:
				var _s string
				_s, err = st.where(v)
				if err != nil {
					return "", err
				}
				s = s + _s
			}

			sep = " , "
		}

		return op + "(" + s + ")", nil
	})
}

func IsValidField(s string) bool {
	for _, ch := range s {
		if !isLetter(ch) && !isDigit(ch) && ch != '_' && ch != '-' && ch != '.' {
			return false
		}
	}

	return true
}

func Quote(s string) string {
	return `'` + strings.Replace(s, `'`, `''`, -1) + `'`
}
