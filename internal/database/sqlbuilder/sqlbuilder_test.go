package sqlbuilder

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
)

func TestBuilderPostgresPlaceholders(t *testing.T) {
	t.Setenv("DATABASE_TYPE", "postgres")

	query, _, err := Builder().
		Select("id").
		From("charities").
		Where(sq.Eq{"id": 42}).
		ToSql()
	if err != nil {
		t.Fatalf("unexpected error building SQL: %v", err)
	}

	if query != "SELECT id FROM charities WHERE id = $1" {
		t.Fatalf("unexpected postgres SQL: %s", query)
	}
}

func TestBuilderQuestionMarkPlaceholders(t *testing.T) {
	t.Setenv("DATABASE_TYPE", "mysql")

	query, _, err := Builder().
		Select("id").
		From("charities").
		Where(sq.Eq{"id": 42}).
		ToSql()
	if err != nil {
		t.Fatalf("unexpected error building SQL: %v", err)
	}

	if query != "SELECT id FROM charities WHERE id = ?" {
		t.Fatalf("unexpected mysql SQL: %s", query)
	}
}

func TestIsMySQL(t *testing.T) {
	t.Setenv("DATABASE_TYPE", "mysql")
	if !IsMySQL() {
		t.Fatal("expected mysql detection to be true")
	}

	t.Setenv("DATABASE_TYPE", "sqlite")
	if IsMySQL() {
		t.Fatal("expected mysql detection to be false")
	}
}
