package rqlParser

import (
	"fmt"
	"io"
	"strings"
)

var IsValueError error = fmt.Errorf("Bloc is a value")

type TokenBloc []TokenString

func (tb TokenBloc) String() (s string) {
	for _, t := range tb {
		s = s + fmt.Sprintf("'%s' ", t.s)
	}
	return
}

type RqlNode struct {
	Op   string
	Args []interface{}
}

type Sort struct {
	by   string
	desc bool
}

type RqlRootNode struct {
	Node   *RqlNode
	limit  string
	offset string
	sorts  []Sort
}

func (r *RqlRootNode) Limit() string {
	return r.limit
}

func (r *RqlRootNode) Offset() string {
	return r.offset
}

func (r *RqlRootNode) Sort() []Sort {
	return r.sorts
}

func parseLimit(n *RqlNode, root *RqlRootNode) (isLimitOp bool) {
	if strings.ToUpper(n.Op) == "LIMIT" {
		root.limit = n.Args[0].(string)
		if len(n.Args) > 1 {
			root.offset = n.Args[1].(string)
		}
		isLimitOp = true
	}
	return
}

func parseSort(n *RqlNode, root *RqlRootNode) (isSortOp bool) {
	if strings.ToUpper(n.Op) == "SORT" {
		for _, s := range n.Args {
			property := s.(string)
			desc := false

			if property[0] == '+' {
				property = property[1:]
			} else if property[0] == '-' {
				desc = true
				property = property[1:]
			}
			root.sorts = append(root.sorts, Sort{by: property, desc: desc})
		}

		isSortOp = true
	}
	return
}

func (r *RqlRootNode) ParseSpecialOps() (err error) {
	if parseLimit(r.Node, r) || parseSort(r.Node, r) {
		r.Node = nil
	} else if strings.ToUpper(r.Node.Op) == "AND" {
		limitIndex := -1
		sortIndex := -1
		for i, c := range r.Node.Args {
			switch n := c.(type) {
			case *RqlNode:
				if parseLimit(n, r) {
					limitIndex = i
				} else if parseSort(n, r) {
					sortIndex = i
				}
			}
		}
		if limitIndex >= 0 {
			if sortIndex > limitIndex {
				sortIndex = sortIndex - 1
			}
			if len(r.Node.Args) == 2 {
				keepIndex := 0
				if limitIndex == 0 {
					keepIndex = 1
				}
				r.Node = r.Node.Args[keepIndex].(*RqlNode)
			} else {
				r.Node.Args = append(r.Node.Args[:limitIndex], r.Node.Args[limitIndex+1:]...)
			}
		}
		if sortIndex >= 0 {
			if len(r.Node.Args) == 2 {
				keepIndex := 0
				if sortIndex == 0 {
					keepIndex = 1
				}
				r.Node = r.Node.Args[keepIndex].(*RqlNode)
			} else {
				r.Node.Args = append(r.Node.Args[:sortIndex], r.Node.Args[sortIndex+1:]...)
			}
		}
	}

	return
}

type Parser struct {
	s *Scanner
}

func NewParser() *Parser {
	return &Parser{s: NewScanner()}
}

func (p *Parser) Parse(r io.Reader) (root *RqlRootNode, err error) {
	var tokenStrings []TokenString
	if tokenStrings, err = p.s.Scan(r); err != nil {
		return nil, err
	}

	root = &RqlRootNode{}

	root.Node, err = parse(tokenStrings)
	if err != nil {
		return nil, err
	}

	root.ParseSpecialOps()

	return
}

func getTokenOp(t Token) string {
	switch t {
	case AMPERSAND, COMMA:
		return "AND"
	case PIPE, SEMI_COLON:
		return "OR"
	}
	return ""
}

func parse(ts []TokenString) (node *RqlNode, err error) {
	var childNode *RqlNode

	childTs := [][]TokenString{}
	node = &RqlNode{}

	if isParenthesisBloc(ts) && findClosingIndex(ts[1:]) == len(ts)-2 {
		ts = ts[1 : len(ts)-1]
	}
	// __printTB("\nParsing : ", ts)

	node.Op, childTs = splitByBasisOp(ts)

	if node.Op == "" || len(childTs) == 1 {
		return getBlocNode(ts)
	}

	for _, c := range childTs {
		childNode, err = parse(c)
		if err != nil {
			if err == IsValueError {
				node.Args = append(node.Args, c[0].s)
			} else {
				return nil, err
			}
		} else {
			node.Args = append(node.Args, childNode)
		}
	}

	return
}

func isTokenInSlice(tokens []Token, tok Token) bool {
	for _, t := range tokens {
		if t == tok {
			return true
		}
	}
	return false
}

func splitByBasisOp(tb []TokenString) (op string, tbs [][]TokenString) {
	matchingToken := ILLEGAL

	prof := 0
	lastIndex := 0

	for i, ts := range tb {
		if ts.t == OPENING_PARENTHESIS {
			prof++
		} else if ts.t == CLOSING_PARENTHESIS {
			prof--
		} else if prof == 0 {
			if (ts.t == AMPERSAND || ts.t == COMMA) && isTokenInSlice([]Token{AMPERSAND, COMMA, ILLEGAL}, matchingToken) {
				tbs = append(tbs, tb[lastIndex:i])
				matchingToken = ts.t
				lastIndex = i + 1
			} else if (ts.t == PIPE || ts.t == SEMI_COLON) && isTokenInSlice([]Token{PIPE, SEMI_COLON, ILLEGAL}, matchingToken) {
				tbs = append(tbs, tb[lastIndex:i])
				matchingToken = ts.t
				lastIndex = i + 1
			}
		}
	}

	tbs = append(tbs, tb[lastIndex:])

	op = getTokenOp(matchingToken)

	return
}

func getBlocNode(tb []TokenString) (*RqlNode, error) {
	n := &RqlNode{}

	if isValue(tb) {
		return nil, IsValueError
	} else if isFuncStyleBloc(tb) {
		var err error

		n.Op = tb[0].s
		n.Args, err = parseFuncArgs(tb[2 : findClosingIndex(tb[2:])+2])
		if err != nil {
			return nil, err
		}
	} else if isSimpleEqualBloc(tb) {
		n.Op = "eq"
		n.Args = []interface{}{tb[0].s, tb[2].s}

	} else if isDoubleEqualBloc(tb) {

		n.Op = tb[2].s
		n.Args = []interface{}{tb[0].s, tb[4].s}

	} else {
		return nil, fmt.Errorf("Unrecognized bloc : " + TokenBloc(tb).String())
	}

	return n, nil
}

func isValue(tb []TokenString) bool {
	return len(tb) == 1 && tb[0].t == IDENT
}

func isParenthesisBloc(tb []TokenString) bool {
	return tb[0].t == OPENING_PARENTHESIS
}

func isFuncStyleBloc(tb []TokenString) bool {
	return (tb[0].t == IDENT) && (tb[1].t == OPENING_PARENTHESIS)
}

func isSimpleEqualBloc(tb []TokenString) bool {
	isSimple := (tb[0].t == IDENT && tb[1].t == EQUAL_SIGN)
	if len(tb) > 3 {
		isSimple = isSimple && tb[3].t != EQUAL_SIGN
	}

	return isSimple
}

func isDoubleEqualBloc(tb []TokenString) bool {
	return tb[0].t == IDENT && tb[1].t == EQUAL_SIGN && tb[2].t == IDENT && tb[3].t == EQUAL_SIGN
}

func parseFuncArgs(tb []TokenString) (args []interface{}, err error) {
	var argTokens [][]TokenString

	indexes := findAllTokenIndexes(tb, COMMA)

	if len(indexes) == 0 {
		argTokens = append(argTokens, tb)
	} else {
		lastIndex := 0
		for _, i := range indexes {
			argTokens = append(argTokens, tb[lastIndex:i])
			lastIndex = i + 1
		}
		argTokens = append(argTokens, tb[lastIndex:])
	}

	for _, ts := range argTokens {
		n, err := parse(ts)
		if err != nil {
			if err == IsValueError {
				args = append(args, ts[0].s)
			} else {
				return args, err
			}
		} else {
			args = append(args, n)
		}
	}

	return
}

func findClosingIndex(tb []TokenString) int {
	i := findTokenIndex(tb, CLOSING_PARENTHESIS)
	return i
}

func findTokenIndex(tb []TokenString, token Token) int {
	prof := 0
	for i, ts := range tb {
		if ts.t == OPENING_PARENTHESIS {
			prof++
		} else if ts.t == CLOSING_PARENTHESIS {
			if prof == 0 && token == CLOSING_PARENTHESIS {
				return i
			}
			prof--
		} else if token == ts.t && prof == 0 {
			return i
		}
	}
	return -1
}

func findAllTokenIndexes(tb []TokenString, token Token) (indexes []int) {
	prof := 0
	for i, ts := range tb {
		if ts.t == OPENING_PARENTHESIS {
			prof++
		} else if ts.t == CLOSING_PARENTHESIS {
			if prof == 0 && token == CLOSING_PARENTHESIS {
				indexes = append(indexes, i)
			}
			prof--
		} else if token == ts.t && prof == 0 {
			indexes = append(indexes, i)
		}
	}
	return
}

func __printTB(s string, tb []TokenString) {
	fmt.Printf("%s%s\n", s, TokenBloc(tb).String())
}
