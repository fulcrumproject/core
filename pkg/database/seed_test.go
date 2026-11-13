// Tests one-time administrative bootstrap against isolated PostgreSQL stores.
// Restarts must preserve credentials rather than recreate a development default.
package database

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSeedOnce(t *testing.T) {
	tdb := newSeedTestDB(t)
	t.Cleanup(func() { tdb.Cleanup(t) })
	// Real startup seeds Core before the separately managed metric schema.
	require.NoError(t, tdb.DB.Migrator().DropTable(&domain.MetricEntry{}))
	start := time.Now()
	require.NoError(t, Seed(tdb.DB))
	var before []domain.Token
	require.NoError(t, tdb.DB.Find(&before).Error)
	require.Len(t, before, 1)
	require.Equal(t, auth.RoleAdmin, before[0].Role)
	require.True(t, before[0].VerifyTokenValue("change-me"))
	require.Equal(t, "Admin Token", before[0].Name)
	require.NotEqual(t, properties.UUID{}, before[0].ID)
	require.WithinDuration(t, start.AddDate(0, 0, 1), before[0].ExpireAt, 5*time.Second)
	require.Nil(t, before[0].ParticipantID)
	require.Nil(t, before[0].AgentID)
	for range 3 {
		require.NoError(t, Seed(tdb.DB))
	}
	var after []domain.Token
	require.NoError(t, tdb.DB.Find(&after).Error)
	require.Equal(t, before, after, "repeated startup must leave every credential field unchanged")
	requireSeedCounts(t, tdb.DB, 1, 1)

	// Ordinary auto-migration on restart must not reset the guard either.
	require.NoError(t, autoMigrate(tdb.DB))
	require.NoError(t, Seed(tdb.DB))
	require.NoError(t, tdb.DB.Find(&after).Error)
	require.Equal(t, before, after)
}

func newSeedTestDB(t *testing.T) *TestDB {
	t.Helper()
	tdb := NewTestDB(t)
	// Isolate statements from NewConnection's configured GORM handle.
	tdb.DB = tdb.DB.Session(&gorm.Session{NewDB: true})
	return tdb
}

func requireSeedCounts(t *testing.T, db *gorm.DB, tokens, guards int64) {
	t.Helper()
	for _, item := range []struct {
		model any
		want  int64
	}{{&domain.Token{}, tokens}, {&bootstrapState{}, guards}} {
		var count int64
		require.NoError(t, db.Model(item.model).Count(&count).Error)
		require.Equal(t, item.want, count)
	}
}

func TestSeedAdoptsTokens(t *testing.T) {
	for _, role := range []auth.Role{auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent} {
		for _, expired := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/expired=%t", role, expired), func(t *testing.T) {
				tdb := newSeedTestDB(t)
				t.Cleanup(func() { tdb.Cleanup(t) })
				expiry := time.Now().Add(time.Hour)
				if expired {
					expiry = time.Now().Add(-time.Hour)
				}
				for range 2 {
					token := &domain.Token{Name: "legacy credential", Role: role, ExpireAt: expiry, HashedValue: domain.HashTokenValue("change-me")}
					// Scope IDs deliberately have no other persisted domain rows:
					// token presence alone must prevent bootstrap, for every role.
					if role != auth.RoleAdmin {
						id := properties.NewUUID()
						token.ParticipantID = &id
					}
					if role == auth.RoleAgent {
						id := properties.NewUUID()
						token.AgentID = &id
					}
					require.NoError(t, tdb.DB.Create(token).Error)
				}
				var before, after []domain.Token
				require.NoError(t, tdb.DB.Order("id").Find(&before).Error)
				for range 2 {
					require.NoError(t, Seed(tdb.DB))
				}
				require.NoError(t, tdb.DB.Order("id").Find(&after).Error)
				require.Equal(t, before, after, "adoption must not reset or deduplicate legacy rows")
				requireSeedCounts(t, tdb.DB, 2, 1)
				require.NoError(t, tdb.DB.Where("1 = 1").Delete(&domain.Token{}).Error)
				require.NoError(t, Seed(tdb.DB))
				requireSeedCounts(t, tdb.DB, 0, 1)
			})
		}
	}
}

func TestSeedAdoptsHistoryWithoutTokens(t *testing.T) {
	for name, data := range map[string]any{
		"domain":       &domain.Participant{Name: "existing participant", Status: domain.ParticipantDisabled},
		"audit":        &domain.Event{Type: domain.EventTypeTokenDeleted, InitiatorType: domain.InitiatorTypeSystem, InitiatorID: "test"},
		"subscription": domain.NewEventSubscription("existing subscriber"),
		"vault":        &vaultSecret{Reference: "existing-reference", EncryptedValue: []byte("synthetic ciphertext")},
		"metric":       &domain.MetricEntry{ID: properties.NewUUID(), ResourceID: "existing resource"},
	} {
		t.Run(name, func(t *testing.T) {
			tdb := newSeedTestDB(t)
			t.Cleanup(func() { tdb.Cleanup(t) })
			require.NoError(t, tdb.DB.Create(data).Error)
			var before string
			require.NoError(t, tdb.DB.Raw("SELECT row_to_json(t)::text FROM (?) t", tdb.DB.Model(data).Select("*")).Scan(&before).Error)
			for range 2 {
				require.NoError(t, Seed(tdb.DB))
			}
			requireSeedCounts(t, tdb.DB, 0, 1)
			var after string
			require.NoError(t, tdb.DB.Raw("SELECT row_to_json(t)::text FROM (?) t", tdb.DB.Model(data).Select("*")).Scan(&after).Error)
			require.Equal(t, before, after)
			// Even subsequent removal of all surviving evidence cannot reset adoption.
			require.NoError(t, tdb.DB.Where("1 = 1").Delete(data).Error)
			require.NoError(t, Seed(tdb.DB))
			requireSeedCounts(t, tdb.DB, 0, 1)
		})
	}
	for _, table := range []string{"agent_type_service_types", "agent_type_infrastructure_types"} {
		t.Run(table, func(t *testing.T) {
			tdb := newSeedTestDB(t)
			t.Cleanup(func() { tdb.Cleanup(t) })
			// Orphaned association evidence must also prevent a new credential.
			require.NoError(t, tdb.DB.Exec("INSERT INTO "+table+" VALUES (?, ?)", properties.NewUUID(), properties.NewUUID()).Error)
			require.NoError(t, Seed(tdb.DB))
			requireSeedCounts(t, tdb.DB, 0, 1)
		})
	}
}

func TestSeedPreservesTokenLifecycle(t *testing.T) {
	tdb := newSeedTestDB(t)
	t.Cleanup(func() { tdb.Cleanup(t) })
	require.NoError(t, Seed(tdb.DB))
	var token domain.Token
	require.NoError(t, tdb.DB.First(&token).Error)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{ID: token.ID, Role: auth.RoleAdmin})
	cmd := domain.NewTokenCommander(NewGormStore(tdb.DB))
	regenerated, err := cmd.Regenerate(ctx, token.ID)
	require.NoError(t, err)
	require.NotEqual(t, domain.HashTokenValue("change-me"), regenerated.HashedValue)
	name, expiry := "secured credential", time.Now().Add(-time.Hour)
	_, err = cmd.Update(ctx, domain.UpdateTokenParams{ID: token.ID, Name: &name, ExpireAt: &expiry})
	require.NoError(t, err)
	var before, after domain.Token
	require.NoError(t, tdb.DB.First(&before).Error)
	require.NoError(t, Seed(tdb.DB))
	require.NoError(t, tdb.DB.First(&after).Error)
	require.Equal(t, before, after)
	requireSeedCounts(t, tdb.DB, 1, 1)
	require.NoError(t, cmd.Delete(ctx, token.ID))
	for range 2 {
		require.NoError(t, Seed(tdb.DB))
	}
	requireSeedCounts(t, tdb.DB, 0, 1)

	// The guard alone is durable evidence even after deletion of audit history.
	require.NoError(t, tdb.DB.Where("1 = 1").Delete(&domain.Event{}).Error)
	require.NoError(t, Seed(tdb.DB))
	requireSeedCounts(t, tdb.DB, 0, 1)
	// Only explicit erasure of guard AND all data/history makes this fixture fresh.
	require.NoError(t, tdb.DB.Where("1 = 1").Delete(&bootstrapState{}).Error)
	require.NoError(t, Seed(tdb.DB))
	requireSeedCounts(t, tdb.DB, 1, 1)
}

func TestSeedAdoptsNativeRevocationBeforeGuard(t *testing.T) {
	tdb := newSeedTestDB(t)
	t.Cleanup(func() { tdb.Cleanup(t) })
	// Simulate a pre-guard store whose final credential was legitimately revoked.
	token := &domain.Token{Name: "legacy final credential", Role: auth.RoleAdmin, HashedValue: domain.HashTokenValue("synthetic"), ExpireAt: time.Now().Add(time.Hour)}
	require.NoError(t, tdb.DB.Create(token).Error)
	ctx := auth.WithIdentity(context.Background(), &auth.Identity{ID: token.ID, Role: auth.RoleAdmin})
	require.NoError(t, domain.NewTokenCommander(NewGormStore(tdb.DB)).Delete(ctx, token.ID))
	requireSeedCounts(t, tdb.DB, 0, 0)
	var history []domain.Event
	require.NoError(t, tdb.DB.Find(&history).Error)
	require.Len(t, history, 1)
	require.Equal(t, domain.EventTypeTokenDeleted, history[0].Type)
	// Upgrade the pre-guard schema using ordinary GORM auto-migration.
	require.NoError(t, tdb.DB.Migrator().DropTable(&bootstrapState{}))
	require.NoError(t, autoMigrate(tdb.DB))
	require.NoError(t, Seed(tdb.DB))
	requireSeedCounts(t, tdb.DB, 0, 1)
	var after []domain.Event
	require.NoError(t, tdb.DB.Find(&after).Error)
	require.Equal(t, history, after)
}

func TestSeedConcurrent(t *testing.T) {
	tdb := newSeedTestDB(t)
	t.Cleanup(func() { tdb.Cleanup(t) })
	for _, adopted := range []bool{false, true} {
		t.Run(fmt.Sprintf("adopted=%t", adopted), func(t *testing.T) {
			require.NoError(t, tdb.DB.Exec("TRUNCATE tokens, bootstrap_states").Error)
			if adopted {
				require.NoError(t, tdb.DB.Create(&domain.Event{Type: domain.EventTypeTokenDeleted, InitiatorType: domain.InitiatorTypeSystem, InitiatorID: "test"}).Error)
			}
			const callers = 12
			start := make(chan struct{})
			errors := make(chan error, callers)
			var wg sync.WaitGroup
			for range callers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					errors <- Seed(tdb.DB.Session(&gorm.Session{NewDB: true}))
				}()
			}
			close(start)
			wg.Wait()
			close(errors)
			for err := range errors {
				require.NoError(t, err)
			}
			want := int64(1)
			if adopted {
				want = 0
			}
			requireSeedCounts(t, tdb.DB, want, 1)
		})
	}
}

func TestSeedRollbackAndRetry(t *testing.T) {
	for _, failure := range []string{"history query", "token insertion", "commit"} {
		t.Run(failure, func(t *testing.T) {
			tdb := newSeedTestDB(t)
			t.Cleanup(func() { tdb.Cleanup(t) })
			switch failure {
			case "history query":
				require.NoError(t, tdb.DB.Exec("ALTER TABLE events RENAME TO hidden_events").Error)
			case "token insertion", "commit":
				require.NoError(t, tdb.DB.Exec(`CREATE FUNCTION reject_bootstrap() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected bootstrap failure'; END $$`).Error)
				trigger := "CREATE TRIGGER reject_bootstrap BEFORE INSERT ON tokens FOR EACH ROW EXECUTE FUNCTION reject_bootstrap()"
				if failure == "commit" {
					trigger = "CREATE CONSTRAINT TRIGGER reject_bootstrap AFTER INSERT ON tokens DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_bootstrap()"
				}
				require.NoError(t, tdb.DB.Exec(trigger).Error)
			}
			require.Error(t, Seed(tdb.DB))
			requireSeedCounts(t, tdb.DB, 0, 0)
			if failure == "history query" {
				require.NoError(t, tdb.DB.Exec("ALTER TABLE hidden_events RENAME TO events").Error)
			} else {
				require.NoError(t, tdb.DB.Exec("DROP TRIGGER reject_bootstrap ON tokens").Error)
				require.NoError(t, tdb.DB.Exec("DROP FUNCTION reject_bootstrap()").Error)
			}
			require.NoError(t, Seed(tdb.DB))
			require.NoError(t, Seed(tdb.DB))
			requireSeedCounts(t, tdb.DB, 1, 1)
		})
	}
}
