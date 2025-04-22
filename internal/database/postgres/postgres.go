package postgres

import (
	"embed"
	"fmt"

	"github.com/Masterminds/squirrel"
)

const (
	TblIssuers              = "issuers"
	TblCredentialsSupported = "credentials_supported"
)

//go:embed migrations
var Migrations embed.FS

func StmtBuilderDollar() squirrel.StatementBuilderType {
	return squirrel.StatementBuilderType{}.PlaceholderFormat(squirrel.Dollar)
}

func StmtBuilderQuestion() squirrel.StatementBuilderType {
	return squirrel.StatementBuilderType{}.PlaceholderFormat(squirrel.Question)
}

func Prepend(tbl, column string) string {
	return fmt.Sprintf("%s.%s", tbl, column)
}

func PrependAll(tbl string, columns ...string) []string {
	out := make([]string, 0)
	for _, column := range columns {
		out = append(out, fmt.Sprintf("%s.%s", tbl, column))
	}

	return out
}
