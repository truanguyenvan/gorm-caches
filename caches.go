package caches

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Caches struct {
	Conf *Config

	queue   *sync.Map
	queryCb func(*gorm.DB)
}

type Config struct {
	Easer      bool
	InstanceId string
	Cacher     Cacher
	Serializer Serializer
	CacheTTL   time.Duration

	// Tables only cache data within given data tables (cache all if empty)
	Tables []string
}

func (c *Caches) Name() string {
	return "gorm:caches"
}

func (c *Caches) Initialize(db *gorm.DB) error {
	if c.Conf == nil {
		c.Conf = &Config{
			Easer:  false,
			Cacher: nil,
		}
	}

	if c.Conf.Easer {
		c.queue = &sync.Map{}
	}

	c.queryCb = db.Callback().Query().Get("gorm:query")

	if err := db.Callback().Query().Replace("gorm:query", c.Query); err != nil {
		return err
	}

	if err := db.Callback().Create().After("*").Register("gorm:cache:after_create", c.AfterCreate); err != nil {
		return err
	}

	if err := db.Callback().Delete().After("*").Register("gorm:cache:after_delete", c.AfterUpdate); err != nil {
		return err
	}

	if err := db.Callback().Update().After("*").Register("gorm:cache:after_update", c.AfterUpdate); err != nil {
		return err
	}

	return nil
}

func (c *Caches) Query(db *gorm.DB) {
	identifier := c.buildIdentifier(db)
	if c.ignoredCache(db) {
		c.ease(db, identifier)
		return
	}

	if c.checkCache(db, identifier) {
		return
	}

	c.ease(db, identifier)
	if db.Error != nil {
		return
	}

	go c.storeInCache(db, identifier)
}

func (c *Caches) AfterUpdate(db *gorm.DB) {
	if db.Error != nil || c.ignoredCache(db) {
		return
	}

	primaryKey := getPrimaryKeyFromWhereClause(db)
	if primaryKey != LIST_KEY {
		// evict cache by detail
		go func() {
			prefixKey := GenCacheKey(c.Conf.InstanceId, db.Statement.Table, primaryKey)
			if err := c.Conf.Cacher.DeleteWithPrefix(db.Statement.Context, prefixKey); err != nil {
				db.Logger.Error(db.Statement.Context, "[AfterUpdate - Delete with key %s] %s", prefixKey, err)
			}
		}()
	}

	// evict cache by list
	go func() {
		prefixKey := GenCacheKey(c.Conf.InstanceId, db.Statement.Table, LIST_KEY)
		if err := c.Conf.Cacher.DeleteWithPrefix(db.Statement.Context, prefixKey); err != nil {
			db.Logger.Error(db.Statement.Context, "[AfterUpdate - Delete with prefix %s] %s", prefixKey, err)
		}
	}()
}

func (c *Caches) AfterCreate(db *gorm.DB) {
	if db.Error != nil || c.ignoredCache(db) {
		return
	}

	// evict cache by list
	go func() {
		prefixKey := GenCacheKey(c.Conf.InstanceId, db.Statement.Table, LIST_KEY)
		if err := c.Conf.Cacher.DeleteWithPrefix(db.Statement.Context, prefixKey); err != nil {
			db.Logger.Error(db.Statement.Context, "[AfterUpdate - Delete with prefix %s] %s", prefixKey, err)
		}
	}()
}

func (c *Caches) ease(db *gorm.DB, identifier string) {
	if c.Conf.Easer == false {
		c.queryCb(db)
		return
	}
	res := ease(&queryTask{
		id:      identifier,
		db:      db,
		queryCb: c.queryCb,
	}, c.queue).(*queryTask)

	if db.Error != nil {
		return
	}

	if res.db.Statement.Dest == db.Statement.Dest {
		return
	}

	q := Query{
		Dest:         db.Statement.Dest,
		RowsAffected: db.Statement.RowsAffected,
	}
	q.replaceOn(res.db)
}

func (c *Caches) checkCache(db *gorm.DB, identifier string) bool {
	if c.Conf.Cacher == nil {
		return false
	}

	var (
		query Query
	)

	res, err := c.Conf.Cacher.Get(db.Statement.Context, identifier)
	if err != nil || res == nil {
		return false
	}

	if err := c.Conf.Serializer.Deserialize(res, &query); err != nil {
		return false
	}

	// binding Statement.Dest
	serializedDest, err := c.Conf.Serializer.Serialize(query.Dest)
	if err != nil {
		return false
	}

	if err := c.Conf.Serializer.Deserialize(serializedDest, &db.Statement.Dest); err != nil {
		return false
	}

	// binding Statement.RowsAffected
	db.Statement.RowsAffected = query.RowsAffected

	return true
}

func (c *Caches) storeInCache(db *gorm.DB, identifier string) {
	if c.Conf.Cacher == nil {
		return
	}

	cachedData, err := c.Conf.Serializer.Serialize(Query{
		Dest:         db.Statement.Dest,
		RowsAffected: db.Statement.RowsAffected,
	})
	if err != nil {
		db.Logger.Error(db.Statement.Context, "[storeInCache - Serialize] %s", err)
		return
	}

	if err := c.Conf.Cacher.Set(db.Statement.Context, identifier, cachedData, c.Conf.CacheTTL); err != nil {
		db.Logger.Error(db.Statement.Context, "[storeInCache - Store] %s", err)
	}
}

func (c *Caches) ctxIgnoredCache(ctx context.Context) bool {
	return ctx.Value(c.Name()) != nil && !ctx.Value(c.Name()).(bool)
}

func (c *Caches) tableIgnoredCache(tableName string) bool {
	return len(c.Conf.Tables) != 0 && ContainString(tableName, c.Conf.Tables)
}

func (c *Caches) ignoredCache(db *gorm.DB) bool {
	return c.Conf.Cacher == nil || c.tableIgnoredCache(db.Statement.Table) || c.ctxIgnoredCache(db.Statement.Context)
}
