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
	if (c.Conf.Easer == false && c.Conf.Cacher == nil) ||
		(db.Statement.Context.Value(c.Name()) != nil && !db.Statement.Context.Value(c.Name()).(bool)) {
		c.queryCb(db)
		return
	}

	identifier := c.buildIdentifier(db)

	if c.checkCache(db, identifier) {
		return
	}

	c.ease(db, identifier)
	if db.Error != nil {
		return
	}

	c.storeInCache(db, identifier)
	if db.Error != nil {
		return
	}
}

func (c *Caches) AfterUpdate(db *gorm.DB) {
	tableName := ""
	if db.Statement.Schema != nil {
		tableName = db.Statement.Schema.Table
	} else {
		tableName = db.Statement.Table
	}

	if db.Error != nil || !c.shouldCache(tableName) {

		return
	}

	ctx := db.Statement.Context
	primaryKey := getPrimaryKeyFromWhereClause(db)
	if primaryKey != LIST_KEY {
		// evict cache by detail
		go func() {
			prefixKey := GenCacheKey(c.Conf.InstanceId, tableName, primaryKey)
			if err := c.Conf.Cacher.DeleteWithPrefix(ctx, prefixKey); err != nil {
				db.Logger.Error(ctx, "[AfterUpdate - Delete with key %s] %s", prefixKey, err)
			}
		}()
	}

	// evict cache by list
	go func() {
		prefixKey := GenCacheKey(c.Conf.InstanceId, tableName, LIST_KEY)
		if err := c.Conf.Cacher.DeleteWithPrefix(ctx, prefixKey); err != nil {
			db.Logger.Error(ctx, "[AfterUpdate - Delete with prefix %s] %s", prefixKey, err)
		}
	}()
}

func (c *Caches) AfterCreate(db *gorm.DB) {
	tableName := ""
	if db.Statement.Schema != nil {
		tableName = db.Statement.Schema.Table
	} else {
		tableName = db.Statement.Table
	}

	if db.Error != nil || !c.shouldCache(tableName) {
		return
	}

	ctx := db.Statement.Context

	// evict cache by list
	go func() {
		prefixKey := GenCacheKey(c.Conf.InstanceId, tableName, LIST_KEY)
		if err := c.Conf.Cacher.DeleteWithPrefix(ctx, prefixKey); err != nil {
			db.Logger.Error(ctx, "[AfterUpdate - Delete with prefix %s] %s", prefixKey, err)
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
		ctx   context.Context
	)
	ctx = db.Statement.Context

	res, err := c.Conf.Cacher.Get(ctx, identifier)
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

	ctx := db.Statement.Context

	cachedData, err := c.Conf.Serializer.Serialize(Query{
		Dest:         db.Statement.Dest,
		RowsAffected: db.Statement.RowsAffected,
	})
	if err != nil {
		db.Logger.Error(ctx, "[storeInCache - Serialize] %s", err)
		return
	}

	if err := c.Conf.Cacher.Set(ctx, identifier, cachedData, c.Conf.CacheTTL); err != nil {
		db.Logger.Error(ctx, "[storeInCache - Store] %s", err)
	}
}

func (c *Caches) shouldCache(tableName string) bool {
	if len(c.Conf.Tables) == 0 {
		return true
	}
	return ContainString(tableName, c.Conf.Tables)
}
