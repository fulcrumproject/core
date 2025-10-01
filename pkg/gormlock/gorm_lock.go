package gormlock

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"
)

var _ gocron.Locker = (*GormLocker)(nil)

type GormLocker struct {
	db            gorm.Interface[CronJobLock]
	worker        string
	ttl           time.Duration
	interval      time.Duration
	jobIdentifier func(ctx context.Context, key string) string

	closed atomic.Bool
}

func NewGormLocker(db *gorm.DB, worker string, options ...LockOption) (*GormLocker, error) {
	if db == nil {
		return nil, ErrGormCantBeNull
	}
	if worker == "" {
		return nil, ErrWorkerIsRequired
	}

	genericDB := gorm.G[CronJobLock](db)

	gl := &GormLocker{
		db:       genericDB,
		worker:   worker,
		ttl:      defaultTTL,
		interval: defaultCleanInterval,
	}
	gl.jobIdentifier = defaultJobIdentifier(defaultPrecision)
	for _, option := range options {
		option(gl)
	}

	ctx := context.Background()

	go func() {
		ticker := time.NewTicker(gl.interval)
		defer ticker.Stop()

		for range ticker.C {
			if gl.closed.Load() {
				return
			}

			gl.cleanExpiredRecords(ctx)
		}
	}()

	return gl, nil
}

func (g *GormLocker) Close() {
	g.closed.Store(true)
}

func (g *GormLocker) Lock(ctx context.Context, key string) (gocron.Lock, error) {
	ji := g.jobIdentifier(ctx, key)

	cjb := &CronJobLock{
		JobName:       key,
		JobIdentifier: ji,
		Worker:        g.worker,
		Status:        StatusRunning,
	}
	tx := g.db.Create(ctx, cjb)
	if tx != nil {
		return nil, tx
	}
	return &gormLock{db: g.db, id: cjb.GetID()}, nil
}

func (g *GormLocker) cleanExpiredRecords(ctx context.Context) {
	g.db.Where("updated_at < ? and status = ?", time.Now().Add(-g.ttl), StatusFinished).Delete(ctx)
}

var _ gocron.Lock = (*gormLock)(nil)

type gormLock struct {
	db gorm.Interface[CronJobLock]
	// id the id that lock a particular job
	id int
}

func (g *gormLock) Unlock(ctx context.Context) error {
	_, err := g.db.Where("id = ?", g.id).Updates(ctx, CronJobLock{Status: StatusFinished})
	return err
}
