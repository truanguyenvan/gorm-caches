package caches

import (
	"testing"

	"gorm.io/gorm"
)

func Test_buildIdentifier(t *testing.T) {
	db := &gorm.DB{}
	caches := &Caches{
		Conf: &Config{
			InstanceId: "123",
		},
	}
	db.Statement = &gorm.Statement{}
	db.Statement.SQL.WriteString("TEST-SQL")
	db.Statement.Vars = append(db.Statement.Vars, "test", 123, 12.3, true, false, []string{"test", "me"})

	actual := caches.buildIdentifier(db)
	expected := "INSTANCE_123:TABLE_:LIST-" + hashKey(`TEST-SQL-[test 123 12.3 true false [test me]]`)
	if actual != expected {
		t.Errorf("buildIdentifier expected to return `%s` but got `%s`", expected, actual)
	}
}
