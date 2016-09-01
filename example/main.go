package main

import (
	"fmt"
	"os"
	"strings"

	rqlParser "github.com/tbaud0n/go-rql-parser"
)

func main() {
	p := rqlParser.NewParser()
	rqlNode, err := p.Parse(strings.NewReader(os.Args[1]))

	if err != nil {
		panic("Error scaning rql : " + err.Error())
	}

	sql, err := rqlParser.NewSqlTranslator(rqlNode).Sql()
	if err != nil {
		panic(err)
	}

	fmt.Println(sql)
}
