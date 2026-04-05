package sqlbuilder

import (
	"os"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

// Builder returns a Squirrel statement builder configured
// for the current DATABASE_TYPE placeholder format.
func Builder() sq.StatementBuilderType {
	if strings.EqualFold(os.Getenv("DATABASE_TYPE"), "postgres") {
		return sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	}
	return sq.StatementBuilder.PlaceholderFormat(sq.Question)
}

// IsMySQL reports whether DATABASE_TYPE is set to mysql.
func IsMySQL() bool {
	return strings.EqualFold(os.Getenv("DATABASE_TYPE"), "mysql")
}
