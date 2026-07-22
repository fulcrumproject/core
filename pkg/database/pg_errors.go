package database

import (
	"errors"
	"regexp"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgNotNullViolation    = "23502"
)

// detailKeyValue matches the "Key (col)=(val)" prefix of a pg constraint Detail.
var detailKeyValue = regexp.MustCompile(`Key \(([^)]+)\)=\(([^)]+)\)`)

// TranslatePgError maps Postgres driver errors to domain errors so the API layer
// classifies them correctly. Unmapped errors are returned unchanged.
func TranslatePgError(err error) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}

	switch pg.Code {
	case pgUniqueViolation:
		field, value, ok := parseKeyDetail(pg)
		if !ok {
			return domain.NewConflictError("resource already exists", nil)
		}
		return domain.NewConflictError("{field} '{value}' already exists", domain.MsgData{
			"field": field,
			"value": value,
		})
	case pgForeignKeyViolation:
		field, value, _ := parseKeyDetail(pg)
		return domain.NewInvalidInputError("referenced {field} '{value}' does not exist", domain.MsgData{
			"field": field,
			"value": value,
		})
	case pgNotNullViolation:
		return domain.NewInvalidInputError("{field} is required", domain.MsgData{
			"field": pg.ColumnName,
		})
	}
	return err
}

// parseKeyDetail extracts the offending column and value from a pg error Detail.
// ok is false when the Detail carries no parseable "Key (col)=(val)" prefix.
func parseKeyDetail(pg *pgconn.PgError) (field, value string, ok bool) {
	if m := detailKeyValue.FindStringSubmatch(pg.Detail); m != nil {
		return m[1], m[2], true
	}
	return "", "", false
}
