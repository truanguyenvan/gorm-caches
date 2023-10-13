package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/truanguyenvan/gorm-caches"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type UserRoleModel struct {
	gorm.Model
	Name string `gorm:"unique"`
}

type UserModel struct {
	gorm.Model
	Name   string
	RoleId uint
	Role   *UserRoleModel `gorm:"foreignKey:role_id;references:id"`
}

type dummyCacher struct {
	store *sync.Map
}

func (c *dummyCacher) init() {
	if c.store == nil {
		c.store = &sync.Map{}
	}
}

func (c *dummyCacher) Get(ctx context.Context, key string) ([]byte, error) {
	c.init()
	val, ok := c.store.Load(key)
	if !ok {
		return nil, nil
	}
	fmt.Println("READ CACHE KEY: ", key)
	return val.([]byte), nil
}

func (c *dummyCacher) Store(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	fmt.Println("SET CACHE KEY: ", key)
	c.init()
	c.store.Store(key, val)
	return nil
}

func (c *dummyCacher) DeleteKey(ctx context.Context, key string) error {
	fmt.Println("DELETE CACHE KEY: ", key)
	c.init()
	c.store.Delete(key)
	return nil
}

func (c *dummyCacher) DeleteKeysWithPrefix(ctx context.Context, keyPrefix string) error {
	fmt.Println("DELETE CACHE PREFIX: ", keyPrefix)
	c.init()
	return nil
}

func main() {
	db, _ := gorm.Open(
		mysql.Open("DNS"),
		&gorm.Config{},
	)
	db = db.Debug()
	cachesPlugin := &caches.Caches{Conf: &caches.Config{
		Cacher:     &dummyCacher{},
		CacheTTL:   5 * time.Minute,
		InstanceId: "ACD12",
		Serializer: caches.JSONSerializer{},
		Easer:      true,
	}}

	_ = db.Use(cachesPlugin)

	_ = db.AutoMigrate(&UserRoleModel{})

	_ = db.AutoMigrate(&UserModel{})

	adminRole := &UserRoleModel{
		Name: "Admin",
	}
	db.FirstOrCreate(adminRole, "Name = ?", "Admin")

	guestRole := &UserRoleModel{
		Name: "Guest",
	}
	db.FirstOrCreate(guestRole, "Name = ?", "Guest")

	db.Save(&UserModel{
		Name: "ktsivkov",
		Role: adminRole,
	})
	db.Save(&UserModel{
		Name: "anonymous",
		Role: guestRole,
	})

	var (
		q1Users []UserModel
		q2Users []UserModel

		q1User UserModel
		q2User UserModel
	)

	q1 := db.Model(&UserModel{}).First(&q1User, 2)
	fmt.Println(fmt.Sprintf("q1User: %+v", q1Users))
	fmt.Println(fmt.Sprintf("q1User- RowsAffected: %+v", q1.RowsAffected))

	q2 := db.Model(&UserModel{}).First(&q2User, 2)
	fmt.Println(fmt.Sprintf("q2Users: %+v", q2Users))
	fmt.Println(fmt.Sprintf("q1Users- RowsAffected: %+v", q2.RowsAffected))

	//ctxIgnorePlugin := context.WithValue(context.Background(), cachesPlugin.Name(), false)
	q1s := db.Model(&UserModel{}).Joins("Role").Find(&q1Users)
	fmt.Println(fmt.Sprintf("q1Users: %+v", q1Users))
	fmt.Println(fmt.Sprintf("q1Users- RowsAffected: %+v", q1s.RowsAffected))

	db.Model(&UserModel{}).Where("id", 1).Update("name", "123")

	q2s := db.Model(&UserModel{}).Joins("Role").Find(&q2Users)
	fmt.Println(fmt.Sprintf("q2Users: %+v", q2Users))
	fmt.Println(fmt.Sprintf("q1Users- RowsAffected: %+v", q2s.RowsAffected))
}
