# GO-RQL-PARSER
[![Build Status](https://travis-ci.org/tbaud0n/go-rql-parser.svg?branch=master)](https://travis-ci.org/tbaud0n/go-rql-parser)    [![Coverage Status](https://coveralls.io/repos/github/tbaud0n/go-rql-parser/badge.svg?branch=master)](https://coveralls.io/github/tbaud0n/go-rql-parser?branch=master)

Small, simple and lightweight library for Go web applications to translate [RQL (Resource Query Language)](http://www.persvr.org/rql/) queries to SQL

## Usage

    query := `and(eq(foo,3),lt(price,10))&sort(+price)`
    
    p := rqlParser.NewParser() // Instanciate a new parser

    // Parse the query and return the root node of the generated tree
	rqlRootNode, err := p.Parse(query) 
	if err != nil {
		panic("Query is not a valid rql : " + err.Error())
	}
	
	// Return the sql string for querying DB
	sql, err := rqlParser.NewSqlTranslator(rqlNode).Sql()
	if err != nil {
		panic(err)
	}
	
	fmt.Println(sql) 
	// Print `WHERE ((foo=3) AND (price < 10)) ORDER BY price

## Supported operators
The library support by default the following RQL operators :
 
 - AND
   - SQL Operator : `AND`
 - OR
   - SQL Operator : `OR` 
 - NE 
	 - SQL Operator : `!=` (When value is `NULL` it is translated to the `IS NOT` SQL operator)
 - EQ
	 - SQL Operator : `=` (When value is `NULL` it is translated to the `IS` SQL operator)
 - LIKE
	 - SQL Operator : `LIKE`
 - MATCH
	- SQL Operator : `ILIKE`
 - GT 
	- SQL Operator : `>`
 - LT
 	- SQL Operator : `<`
 - GE
 	- SQL Operator : `>=`
 - LE
 	- SQL Operator : `<=`
 - SUM
 	- SQL Operator : `SUM`
 - NOT
 	- SQL Operator : `NOT`

## Append supported operators
It's easy to handle new operators by simply adding a `TranslatorOpFunc` to the `SQLTranslator` :
    
    query := `or(eq(foo,42),between(price,10,100))`
    rqlRootNode, err := p.Parse(query) 
    if err != nil { 
      panic("Query is not a valid rql : " + err.Error()) 
    }

    sqlTranslator := rqlParser.NewSqlTranslator(rqlNode)
    
    // Before translating the node, just add the custom operator
    // let's define a new function to handle beetween operator
	betweenTranslatorFunc := func(n *rqlParser.RqlNode) (s string, err error) {
		var (
			min, max int64
			field    string
		)

		if len(n.Args) != 3 {
			err = fmt.Errorf("between operator require 3 arguments")
			return
		}

		for i, a := range n.Args {
			if v, ok := a.(string); ok {
				switch i {
				case 0:
					if rqlParser.IsValidField(v) {
						field = v
					} else {
						return "", fmt.Errorf("First argument must be a valid field name (arg: %s)", v)
					}
				case 1, 2:
					num, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						return "", fmt.Errorf("Argument n°%d must be an integer (arg: %s)", i, v)
					}
					if i == 1 {
						min = num
					} else {
						max = num
					}
				}
			} else {
				err = fmt.Errorf("Between example function only support string arguments")
				return
			}
		}

		s = fmt.Sprintf("(%s >= %d) AND (%s <= %d)", field, min, field, max)

		return
	}
    
    sqlTranslator.SetOpFunc(`between`, rqlParser.TranslatorOpFunc(betweenTranslatorFunc))
    s, err := sqlTranslator.Sql() 
    if err != nil { 
      panic(err) 
    }
    fmt.Println(s) // Will print : "WHERE ((foo=42) OR ((price >= 10) AND (price <= 100)))"

## Contributions

Any contribution is welcome. 

There is still many improvements that should be added to this library :
- Support type casting (ex: `string:42` to force the SQLTranslator to handle 42 as a string)
- Create a per database architecture translator as all databases doesn't support the same operators.
