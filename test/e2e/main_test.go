package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/database"
)

func TestE2E(t *testing.T) {
	tdb := database.NewTestDB(t)
	t.Cleanup(func() { tdb.Cleanup(t) })

	mustSeed(t, tdb.DB)

}
