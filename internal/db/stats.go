package db

import (
	"expvar"
	"sync"

	"github.com/jmoiron/sqlx"
)

var (
	dbStatsOnce sync.Once
	dbStatsVar  *expvar.Map
)

func initDBStatsVars() {
	dbStatsOnce.Do(func() {
		dbStatsVar = expvar.NewMap("db_pool")
	})
}

func PublishStats(db *sqlx.DB) {
	if db == nil {
		return
	}

	initDBStatsVars()

	dbStatsVar.Set("max_open_connections", expvar.Func(func() any {
		return db.Stats().MaxOpenConnections
	}))
	dbStatsVar.Set("open_connections", expvar.Func(func() any {
		return db.Stats().OpenConnections
	}))
	dbStatsVar.Set("in_use", expvar.Func(func() any {
		return db.Stats().InUse
	}))
	dbStatsVar.Set("idle", expvar.Func(func() any {
		return db.Stats().Idle
	}))
	dbStatsVar.Set("wait_count", expvar.Func(func() any {
		return db.Stats().WaitCount
	}))
	dbStatsVar.Set("wait_duration_ns", expvar.Func(func() any {
		return db.Stats().WaitDuration.Nanoseconds()
	}))
	dbStatsVar.Set("max_idle_closed", expvar.Func(func() any {
		return db.Stats().MaxIdleClosed
	}))
	dbStatsVar.Set("max_idle_time_closed", expvar.Func(func() any {
		return db.Stats().MaxIdleTimeClosed
	}))
	dbStatsVar.Set("max_lifetime_closed", expvar.Func(func() any {
		return db.Stats().MaxLifetimeClosed
	}))
}
