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

func (st *SqlTranslator) Sql() (string, error) {
	sql, err := st.Where()
	if err != nil {
		return sql, err
	}

	sql = sql + st.Sort() + st.Limit() + st.Offset()

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
	st.SetOpFunc("LIKE", st.GetFieldValueTranslatorFunc("LIKE", starToPercentFunc))
	st.SetOpFunc("MATCH", st.GetFieldValueTranslatorFunc("ILIKE", starToPercentFunc))
	st.SetOpFunc("GT", st.GetFieldValueTranslatorFunc(">", nil))
	st.SetOpFunc("LT", st.GetFieldValueTranslatorFunc("<", nil))
	st.SetOpFunc("GE", st.GetFieldValueTranslatorFunc(">=", nil))
	st.SetOpFunc("LE", st.GetFieldValueTranslatorFunc("<=", nil))
	st.SetOpFunc("NE", st.GetFieldValueTranslatorFunc("!=", nil))
	st.SetOpFunc("SUM", st.GetOpFirstTranslatorFunc("SUM", nil))
	st.SetOpFunc("NOT", st.GetOpFirstTranslatorFunc("NOT", nil))
	st.SetOpFunc("EQ", TranslatorOpFunc(func(n *RqlNode) (string, error) {
		op := `=`
		field := n.Args[0].(string)

		if !IsValidField(field) {
			return ``, fmt.Errorf("Invalid field name : %s", field)
		}

		value, err := url.QueryUnescape(n.Args[1].(string))
		if err != nil {
			return "", err
		}

		if value == `null` {
			return fmt.Sprintf("%s IS NULL", field), nil
		}

		value = Quote(value)

		return fmt.Sprintf("%s %s %s", field, op, value), nil
	}))

	return
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

		// field := n.Args[0].(string)

		// if !IsValidField(field) {
		// 	return ``, fmt.Errorf("Invalid field name : %s", field)
		// }

		// value, err := url.QueryUnescape(n.Args[1].(string))
		// if err != nil {
		// 	return "", err
		// }

		// if valueAlterFunc != nil {
		// 	value, err = valueAlterFunc(value)
		// 	if err != nil {
		// 		return "", err
		// 	}
		// } else {
		// 	value = Quote(value)
		// }

		// return fmt.Sprintf("%s %s %s", field, op, value), nil
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
