package caches

import (
	"encoding/base64"
	"fmt"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"reflect"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

const (
	CACHE_PATTERN = "INSTANCE_%s:TABLE_%s:%s" //instanceId:tableName:keyCache

	LIST_KEY = "LIST"
)

func hashKey(key string) string {
	return base64.StdEncoding.EncodeToString([]byte(key))
}

func (c *Caches) buildIdentifier(db *gorm.DB) string {
	// Build query identifier,
	//	for that reason we need to compile all arguments into a string
	//	and concat them with the SQL query itself
	var (
		tableName string
		keys      []string
	)

	if db.Statement.Schema != nil {
		tableName = db.Statement.Schema.Table
	} else {
		tableName = db.Statement.Table
	}

	primaryKey := getPrimaryKeyFromWhereClause(db)
	if primaryKey != "" {
		keys = append(keys, primaryKey)
	}

	callbacks.BuildQuerySQL(db)
	keys = append(keys, hashKey(fmt.Sprintf("%s-%s", db.Statement.SQL.String(), fmt.Sprintf("%v", db.Statement.Vars))))

	return GenCacheKey(c.Conf.InstanceId, tableName, strings.Join(keys, "-"))
}

func GenCacheKey(instanceId, tableName, key string) string {
	return fmt.Sprintf(CACHE_PATTERN, instanceId, tableName, key)
}

func GenCachePrefix(instanceId string, tableName string) string {
	return fmt.Sprintf(CACHE_PATTERN, instanceId, tableName, "")
}

func getPrimaryKeyFromWhereClause(db *gorm.DB) string {
	primaryKeys := make([]string, 0)
	claLimit, ok := db.Statement.Clauses["LIMIT"]
	if !ok {
		return LIST_KEY
	}

	limit, ok := claLimit.Expression.(clause.Limit)
	if !ok || *limit.Limit != 1 {
		return LIST_KEY
	}

	claWhere, ok := db.Statement.Clauses["WHERE"]
	if !ok {
		return LIST_KEY
	}

	dbName := ""
	for _, field := range db.Statement.Schema.Fields {
		if field.PrimaryKey {
			dbName = field.DBName
			break
		}
	}
	if len(dbName) == 0 {
		return LIST_KEY
	}

	where, ok := claWhere.Expression.(clause.Where)
	if !ok {
		return LIST_KEY
	}

	for _, expr := range where.Exprs {
		eqExpr, ok := expr.(clause.Eq)
		if ok {
			if eqExpr.Column == clause.PrimaryKey || getColNameFromColumn(eqExpr.Column) == dbName {
				primaryKeys = append(primaryKeys, fmt.Sprintf("%v", eqExpr.Value))
				continue
			}
		}

		inExpr, ok := expr.(clause.IN)
		if ok {
			if getColNameFromColumn(inExpr.Column) == clause.PrimaryKey || getColNameFromColumn(inExpr.Column) == dbName {
				for _, val := range inExpr.Values {
					primaryKeys = append(primaryKeys, fmt.Sprintf("%v", val))
				}
				continue
			}
		}

		exprStruct, ok := expr.(clause.Expr)
		if ok {
			ttype := getExprType(exprStruct)
			if ttype == "in" || ttype == "eq" {
				colName := getColNameFromExpr(exprStruct, ttype)
				splitedColName := strings.Split(colName, ".")
				fieldName := splitedColName[len(splitedColName)-1]
				if fieldName == dbName {
					pKeys := getPrimaryKeysFromExpr(exprStruct, ttype)
					primaryKeys = append(primaryKeys, pKeys...)
				}
			}
		}
	}

	if len(primaryKeys) == 0 {
		return LIST_KEY
	}
	return strings.Join(primaryKeys, "_")
}

func getColNameFromColumn(col interface{}) string {
	switch v := col.(type) {
	case string:
		return v
	case clause.Column:
		return v.Name
	default:
		return ""
	}
}

func getColNameFromExpr(expr clause.Expr, ttype string) string {
	sql := strings.Replace(strings.ToLower(expr.SQL), " ", "", -1)
	if ttype == "in" {
		fields := strings.Split(sql, "in")
		return fields[0]
	} else if ttype == "eq" {
		fields := strings.Split(sql, "=")
		return fields[0]
	}
	return ""
}

func getPrimaryKeysFromExpr(expr clause.Expr, ttype string) []string {
	sql := strings.Replace(strings.ToLower(expr.SQL), " ", "", -1)

	primaryKeys := make([]string, 0)

	if ttype == "in" {
		fields := strings.Split(sql, "in")
		if len(fields) == 2 {
			if fields[1][0] == '(' && fields[1][len(fields[1])-1] == ')' {
				idStr := fields[1][1 : len(fields[1])-1]
				ids := strings.Split(idStr, ",")
				for _, id := range ids {
					if id == "?" {
						for _, vvar := range expr.Vars {
							keys := extractStringsFromVar(vvar)
							primaryKeys = append(primaryKeys, keys...)
						}
						break
					}
					number, err := strconv.ParseInt(id, 10, 64)
					if err == nil {
						primaryKeys = append(primaryKeys, strconv.FormatInt(number, 10))
					}
				}
			} else if fields[1] == "(?)" {
				for _, val := range expr.Vars {
					primaryKeys = append(primaryKeys, fmt.Sprintf("%v", val))
				}
			}
		}
	} else if ttype == "eq" {
		fields := strings.Split(sql, "=")
		if len(fields) == 2 {
			_, err := strconv.ParseInt(fields[1], 10, 64)
			if err == nil {
				primaryKeys = append(primaryKeys, fields[1])
			} else if fields[1] == "?" {
				for _, val := range expr.Vars {
					primaryKeys = append(primaryKeys, fmt.Sprintf("%v", val))
				}
			}
		}
	}
	return primaryKeys
}

func extractStringsFromVar(v interface{}) []string {
	noPtrValue := reflect.Indirect(reflect.ValueOf(v))
	switch noPtrValue.Kind() {
	case reflect.Slice, reflect.Array:
		ans := make([]string, 0)
		for i := 0; i < noPtrValue.Len(); i++ {
			obj := reflect.Indirect(noPtrValue.Index(i))
			ans = append(ans, fmt.Sprintf("%v", obj))
		}
		return ans
	case reflect.String:
		return []string{fmt.Sprintf("%s", noPtrValue.Interface())}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []string{fmt.Sprintf("%d", noPtrValue.Interface())}
	}
	return nil
}

func getExprType(expr clause.Expr) string {
	// delete spaces
	sql := strings.Replace(strings.ToLower(expr.SQL), " ", "", -1)

	// see if sql has more than one clause
	hasConnector := strings.Contains(sql, "and") || strings.Contains(sql, "or")

	if strings.Contains(sql, "=") && !hasConnector {
		// possibly "id=?" or "id=123"
		fields := strings.Split(sql, "=")
		if len(fields) == 2 {
			_, isNumberErr := strconv.ParseInt(fields[1], 10, 64)
			if fields[1] == "?" || isNumberErr == nil {
				return "eq"
			}
		}
	} else if strings.Contains(sql, "in") && !hasConnector {
		// possibly "idIN(?)"
		fields := strings.Split(sql, "in")
		if len(fields) == 2 {
			if len(fields[1]) > 1 && fields[1][0] == '(' && fields[1][len(fields[1])-1] == ')' {
				return "in"
			}
		}
	}
	return "other"
}
