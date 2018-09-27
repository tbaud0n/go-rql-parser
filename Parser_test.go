package rqlParser

import (
	"strings"
	"testing"
)

type Test struct {
	Name                string // Name of the test
	RQL                 string // Input RQL query
	SQL                 string // Expected Output SQL
	WantParseError      bool   // Test should raise an error when parsing the RQL query
	WantTranslatorError bool   // Test should raise an error when translating to SQL
}

func (test *Test) Run(t *testing.T) {
	p := NewParser()

	rqlNode, err := p.Parse(strings.NewReader(test.RQL))
	if test.WantParseError != (err != nil) {
		t.Fatalf("(%s) Expecting error :%v\nGot error : %v", test.Name, test.WantParseError, err)
	}

	sqlTranslator := NewSqlTranslator(rqlNode)
	s, err := sqlTranslator.Sql()
	if test.WantTranslatorError != (err != nil) {
		t.Fatalf("(%s) Expecting error :%v\nGot error : %v \n\tSQL = %s", test.Name, test.WantTranslatorError, err, s)
	}

	if s != test.SQL {
		t.Fatalf("(%s) Translated SQL doesn’t match the expected one %s vs %s", test.Name, s, test.SQL)
	}
}

var tests = []Test{
	{
		Name:                `Basic translation with double equal operators`,
		RQL:                 `and(foo=eq=42,price=gt=10)`,
		SQL:                 `WHERE ((foo = 42) AND (price > 10))`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Basic translation with func style operators`,
		RQL:                 `and(eq(foo,42),gt(price,10),not(disabled))`,
		SQL:                 `WHERE ((foo = 42) AND (price > 10) AND NOT(disabled))`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Basic translation with func simple equal operators`,
		RQL:                 `foo=42&price=10`,
		SQL:                 `WHERE ((foo = 42) AND (price = 10))`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Sort and limit`,
		RQL:                 `eq(foo,42)&sort(+price,-length)&limit(10,20)`,
		SQL:                 `WHERE (foo = 42) ORDER BY price, length DESC LIMIT 10 OFFSET 20`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Sort only`,
		RQL:                 `sort(-price)`,
		SQL:                 ` ORDER BY price DESC`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `LIKE empty string`,
		RQL:                 `foo=like=`,
		SQL:                 `WHERE (foo LIKE '')`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Mixed style translation`,
		RQL:                 `((eq(foo,42)&gt(price,10))|price=ge=500)&disabled=eq=false`,
		SQL:                 `WHERE ((((foo = 42) AND (price > 10)) OR (price >= 500)) AND (disabled IS FALSE))`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Try a simple SQL injection`,
		RQL:                 `foo=like=toto%27%3BSELECT%20column%20IN%20table`,
		SQL:                 `WHERE (foo LIKE 'toto'';SELECT column IN table')`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Empty RQL`,
		RQL:                 ``,
		SQL:                 ``,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Invalid RQL query (Unmanaged RQL operator)`,
		RQL:                 `foo=missing_operator=42`,
		SQL:                 ``,
		WantParseError:      false,
		WantTranslatorError: true,
	},
	{
		Name:                `Invalid RQL query (Unescaped character)`,
		RQL:                 `like(foo,hello world)`,
		SQL:                 ``,
		WantParseError:      true,
		WantTranslatorError: false,
	},
	{
		Name:                `Invalid RQL query (Missing comma)`,
		RQL:                 `and(not(test),eq(foo,toto)gt(price,10))`,
		SQL:                 ``,
		WantParseError:      true,
		WantTranslatorError: false,
	},
	{
		Name:                `Invalid RQL query (Invalid field name)`,
		RQL:                 `eq(foo%20tot,42)`,
		SQL:                 ``,
		WantParseError:      false,
		WantTranslatorError: true,
	},
	{
		Name:                `Invalid RQL query (Invalid field name 2)`,
		RQL:                 `eq(foo*,toto)`,
		SQL:                 ``,
		WantParseError:      false,
		WantTranslatorError: true,
	},
}

func TestParser(t *testing.T) {
	for _, test := range tests {
		test.Run(t)
	}
}
