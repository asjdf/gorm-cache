package cache

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// getPrimaryKeyFields 获取所有主键字段（支持联合主键）
// 返回的字段已按主键顺序排列
func getPrimaryKeyFields(s *schema.Schema) []*schema.Field {
	if s == nil {
		return nil
	}
	// 使用Schema的PrimaryFields，它已经按顺序排列
	if len(s.PrimaryFields) > 0 {
		return s.PrimaryFields
	}
	// 如果没有PrimaryFields，则从Fields中查找（兼容旧版本）
	primaryKeyFields := make([]*schema.Field, 0)
	for _, field := range s.Fields {
		if field.PrimaryKey {
			primaryKeyFields = append(primaryKeyFields, field)
		}
	}
	return primaryKeyFields
}

// getPrimaryKeysFromWhereClause 从WHERE子句中提取主键值，支持联合主键
// 返回格式：对于单个主键，返回["value1", "value2"]（多个记录）
// 对于联合主键，返回["field1:value1:field2:value2", "field1:value3:field2:value4"]（多个记录）
// 注意：联合主键的key格式为按字段顺序排列的值，用":"分隔
func getPrimaryKeysFromWhereClause(db *gorm.DB) []string {
	cla, ok := db.Statement.Clauses["WHERE"]
	if !ok {
		return nil
	}
	where, ok := cla.Expression.(clause.Where)
	if !ok {
		return nil
	}
	if db.Statement.Schema == nil {
		return nil
	}

	primaryKeyFields := getPrimaryKeyFields(db.Statement.Schema)
	if len(primaryKeyFields) == 0 {
		return nil
	}

	// 收集WHERE子句中的字段值
	fieldValuesMap := make(map[string][]string) // key: fieldName, value: []values
	for _, expr := range where.Exprs {
		eqExpr, ok := expr.(clause.Eq)
		if ok {
			fieldName := getColNameFromColumn(eqExpr.Column)
			fieldValuesMap[fieldName] = append(fieldValuesMap[fieldName], fmt.Sprintf("%v", eqExpr.Value))
			continue
		}
		inExpr, ok := expr.(clause.IN)
		if ok {
			fieldName := getColNameFromColumn(inExpr.Column)
			values := make([]string, 0, len(inExpr.Values))
			for _, val := range inExpr.Values {
				values = append(values, fmt.Sprintf("%v", val))
			}
			fieldValuesMap[fieldName] = append(fieldValuesMap[fieldName], values...)
			continue
		}
		exprStruct, ok := expr.(clause.Expr)
		if ok {
			ttype := getExprType(exprStruct)
			if ttype == "in" || ttype == "eq" {
				fieldName := getColNameFromExpr(exprStruct, ttype)
				pKeys := getPrimaryKeysFromExpr(exprStruct, ttype)
				fieldValuesMap[fieldName] = append(fieldValuesMap[fieldName], pKeys...)
			}
		}
	}

	// 检查是否所有主键字段都有值
	for _, field := range primaryKeyFields {
		if len(fieldValuesMap[field.DBName]) == 0 {
			return nil // 缺少某个主键字段的值
		}
	}

	// 生成主键key列表
	// 对于单个主键：直接返回所有值
	if len(primaryKeyFields) == 1 {
		return uniqueStringSlice(fieldValuesMap[primaryKeyFields[0].DBName])
	}

	// 对于联合主键：需要组合所有字段的值
	// 先获取每个字段的值列表长度，取最小值（因为可能有IN查询）
	maxLen := len(fieldValuesMap[primaryKeyFields[0].DBName])
	for _, field := range primaryKeyFields[1:] {
		if len(fieldValuesMap[field.DBName]) < maxLen {
			maxLen = len(fieldValuesMap[field.DBName])
		}
	}

	// 生成联合主键key
	primaryKeys := make([]string, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		keyParts := make([]string, 0, len(primaryKeyFields))
		for _, field := range primaryKeyFields {
			values := fieldValuesMap[field.DBName]
			if i < len(values) {
				keyParts = append(keyParts, values[i])
			} else {
				// 如果某个字段的值不够，使用最后一个值
				keyParts = append(keyParts, values[len(values)-1])
			}
		}
		primaryKeys = append(primaryKeys, strings.Join(keyParts, ":"))
	}

	return uniqueStringSlice(primaryKeys)
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

// hasOtherClauseExceptPrimaryField 检查WHERE子句中是否有除了主键字段之外的其他条件
// 支持联合主键
func hasOtherClauseExceptPrimaryField(db *gorm.DB) bool {
	cla, ok := db.Statement.Clauses["WHERE"]
	if !ok {
		return false
	}
	where, ok := cla.Expression.(clause.Where)
	if !ok {
		return false
	}
	if db.Statement.Schema == nil {
		return true
	}

	primaryKeyFields := getPrimaryKeyFields(db.Statement.Schema)
	if len(primaryKeyFields) == 0 {
		return true // 没有主键，返回true跳过缓存
	}

	// 构建主键字段名集合
	primaryKeyFieldSet := make(map[string]struct{}, len(primaryKeyFields))
	for _, field := range primaryKeyFields {
		primaryKeyFieldSet[field.DBName] = struct{}{}
	}

	// 检查每个表达式
	for _, expr := range where.Exprs {
		eqExpr, ok := expr.(clause.Eq)
		if ok {
			fieldName := getColNameFromColumn(eqExpr.Column)
			if _, isPrimaryKey := primaryKeyFieldSet[fieldName]; !isPrimaryKey {
				return true
			}
			continue
		}
		inExpr, ok := expr.(clause.IN)
		if ok {
			fieldName := getColNameFromColumn(inExpr.Column)
			if _, isPrimaryKey := primaryKeyFieldSet[fieldName]; !isPrimaryKey {
				return true
			}
			continue
		}
		exprStruct, ok := expr.(clause.Expr)
		if ok {
			ttype := getExprType(exprStruct)
			if ttype == "in" || ttype == "eq" {
				fieldName := getColNameFromExpr(exprStruct, ttype)
				if _, isPrimaryKey := primaryKeyFieldSet[fieldName]; !isPrimaryKey {
					return true
				}
				continue
			}
			return true
		}
		// 其他类型的表达式，视为有其他条件
		return true
	}
	return false
}

// getUniqueIndexFields 获取指定unique索引的字段列表
func getUniqueIndexFields(s *schema.Schema, indexName string) []*schema.Field {
	if s == nil {
		return nil
	}
	allIndexes := s.ParseIndexes()
	for _, index := range allIndexes {
		if index.Name == indexName && index.Class == "UNIQUE" {
			fields := make([]*schema.Field, 0, len(index.Fields))
			for _, fieldOption := range index.Fields {
				if fieldOption.Field != nil {
					fields = append(fields, fieldOption.Field)
				}
			}
			return fields
		}
	}
	return nil
}

// getAllUniqueIndexes 获取所有unique索引
func getAllUniqueIndexes(s *schema.Schema) map[string]*schema.Index {
	if s == nil {
		return nil
	}
	allIndexes := s.ParseIndexes()
	uniqueIndexes := make(map[string]*schema.Index)
	for _, index := range allIndexes {
		if index.Class == "UNIQUE" {
			uniqueIndexes[index.Name] = index
		}
	}
	if len(uniqueIndexes) == 0 {
		return nil
	}
	return uniqueIndexes
}

// getUniqueKeysFromWhereClause 从WHERE子句中提取unique键值
// 返回: map[uniqueIndexName][]string，每个unique索引对应一个key列表
func getUniqueKeysFromWhereClause(db *gorm.DB) map[string][]string {
	cla, ok := db.Statement.Clauses["WHERE"]
	if !ok {
		return nil
	}
	where, ok := cla.Expression.(clause.Where)
	if !ok {
		return nil
	}
	if db.Statement.Schema == nil {
		return nil
	}

	uniqueIndexes := getAllUniqueIndexes(db.Statement.Schema)
	if len(uniqueIndexes) == 0 {
		return nil
	}

	// 收集WHERE子句中的字段值
	fieldValuesMap := make(map[string][]string) // key: fieldName, value: []values
	for _, expr := range where.Exprs {
		eqExpr, ok := expr.(clause.Eq)
		if ok {
			fieldName := getColNameFromColumn(eqExpr.Column)
			fieldValuesMap[fieldName] = append(fieldValuesMap[fieldName], fmt.Sprintf("%v", eqExpr.Value))
			continue
		}
		inExpr, ok := expr.(clause.IN)
		if ok {
			fieldName := getColNameFromColumn(inExpr.Column)
			values := make([]string, 0, len(inExpr.Values))
			for _, val := range inExpr.Values {
				values = append(values, fmt.Sprintf("%v", val))
			}
			fieldValuesMap[fieldName] = append(fieldValuesMap[fieldName], values...)
			continue
		}
		exprStruct, ok := expr.(clause.Expr)
		if ok {
			ttype := getExprType(exprStruct)
			if ttype == "in" || ttype == "eq" {
				fieldName := getColNameFromExpr(exprStruct, ttype)
				pKeys := getPrimaryKeysFromExpr(exprStruct, ttype)
				fieldValuesMap[fieldName] = append(fieldValuesMap[fieldName], pKeys...)
			}
		}
	}

	// 检查每个unique索引，看是否所有字段都有值
	result := make(map[string][]string)
	for indexName, index := range uniqueIndexes {
		// 检查该unique索引的所有字段是否都有值
		allFieldsHaveValues := true
		for _, fieldOption := range index.Fields {
			if fieldOption.Field == nil {
				allFieldsHaveValues = false
				break
			}
			if len(fieldValuesMap[fieldOption.Field.DBName]) == 0 {
				allFieldsHaveValues = false
				break
			}
		}
		if !allFieldsHaveValues {
			continue
		}

		// 生成unique键key列表
		if len(index.Fields) == 1 {
			// 单个字段的unique键
			if index.Fields[0].Field != nil {
				result[indexName] = uniqueStringSlice(fieldValuesMap[index.Fields[0].Field.DBName])
			}
		} else {
			// 联合unique键
			if len(index.Fields) == 0 {
				continue
			}
			firstField := index.Fields[0].Field
			if firstField == nil {
				continue
			}
			maxLen := len(fieldValuesMap[firstField.DBName])
			for _, fieldOption := range index.Fields[1:] {
				if fieldOption.Field == nil {
					continue
				}
				if len(fieldValuesMap[fieldOption.Field.DBName]) < maxLen {
					maxLen = len(fieldValuesMap[fieldOption.Field.DBName])
				}
			}

			uniqueKeys := make([]string, 0, maxLen)
			for i := 0; i < maxLen; i++ {
				keyParts := make([]string, 0, len(index.Fields))
				for _, fieldOption := range index.Fields {
					if fieldOption.Field == nil {
						continue
					}
					values := fieldValuesMap[fieldOption.Field.DBName]
					if i < len(values) {
						keyParts = append(keyParts, values[i])
					} else if len(values) > 0 {
						keyParts = append(keyParts, values[len(values)-1])
					}
				}
				if len(keyParts) == len(index.Fields) {
					uniqueKeys = append(uniqueKeys, strings.Join(keyParts, ":"))
				}
			}
			if len(uniqueKeys) > 0 {
				result[indexName] = uniqueStringSlice(uniqueKeys)
			}
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// hasOtherClauseExceptUniqueField 检查WHERE子句中是否有除了指定unique索引字段之外的其他条件
func hasOtherClauseExceptUniqueField(db *gorm.DB, uniqueIndexName string) bool {
	cla, ok := db.Statement.Clauses["WHERE"]
	if !ok {
		return false
	}
	where, ok := cla.Expression.(clause.Where)
	if !ok {
		return false
	}
	if db.Statement.Schema == nil {
		return true
	}

	uniqueFields := getUniqueIndexFields(db.Statement.Schema, uniqueIndexName)
	if len(uniqueFields) == 0 {
		return true
	}

	// 构建unique字段名集合
	uniqueFieldSet := make(map[string]struct{}, len(uniqueFields))
	for _, field := range uniqueFields {
		uniqueFieldSet[field.DBName] = struct{}{}
	}

	// 检查每个表达式
	for _, expr := range where.Exprs {
		eqExpr, ok := expr.(clause.Eq)
		if ok {
			fieldName := getColNameFromColumn(eqExpr.Column)
			if _, isUniqueField := uniqueFieldSet[fieldName]; !isUniqueField {
				return true
			}
			continue
		}
		inExpr, ok := expr.(clause.IN)
		if ok {
			fieldName := getColNameFromColumn(inExpr.Column)
			if _, isUniqueField := uniqueFieldSet[fieldName]; !isUniqueField {
				return true
			}
			continue
		}
		exprStruct, ok := expr.(clause.Expr)
		if ok {
			ttype := getExprType(exprStruct)
			if ttype == "in" || ttype == "eq" {
				fieldName := getColNameFromExpr(exprStruct, ttype)
				if _, isUniqueField := uniqueFieldSet[fieldName]; !isUniqueField {
					return true
				}
				continue
			}
			return true
		}
		return true
	}
	return false
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

// getObjectsAfterLoad 从查询结果中提取主键和对象，支持联合主键
// 返回的主键格式：对于单个主键，返回["value1", "value2"]
// 对于联合主键，返回["value1:value2", "value3:value4"]（按主键字段顺序）
func getObjectsAfterLoad(db *gorm.DB) (primaryKeys []string, objects []interface{}) {
	primaryKeys = make([]string, 0)
	values := make([]reflect.Value, 0)

	destValue := reflect.Indirect(reflect.ValueOf(db.Statement.Dest))
	switch destValue.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < destValue.Len(); i++ {
			elem := destValue.Index(i)
			values = append(values, elem)
		}
	case reflect.Struct:
		values = append(values, destValue)
	}

	if db.Statement.Schema == nil {
		// 没有schema，无法提取主键，只返回对象
		objects = make([]interface{}, 0, len(values))
		for _, elemValue := range values {
			objects = append(objects, elemValue.Interface())
		}
		return primaryKeys, objects
	}

	primaryKeyFields := getPrimaryKeyFields(db.Statement.Schema)
	if len(primaryKeyFields) == 0 {
		// 没有主键字段，只返回对象
		objects = make([]interface{}, 0, len(values))
		for _, elemValue := range values {
			objects = append(objects, elemValue.Interface())
		}
		return primaryKeys, objects
	}

	objects = make([]interface{}, 0, len(values))
	for _, elemValue := range values {
		// 提取主键值
		if len(primaryKeyFields) == 1 {
			// 单个主键
			valueOf := primaryKeyFields[0].ValueOf
			if valueOf != nil {
				primaryKey, isZero := valueOf(context.Background(), elemValue)
				if isZero {
					continue
				}
				primaryKeys = append(primaryKeys, fmt.Sprintf("%v", primaryKey))
			}
		} else {
			// 联合主键
			keyParts := make([]string, 0, len(primaryKeyFields))
			allZero := true
			for _, field := range primaryKeyFields {
				valueOf := field.ValueOf
				if valueOf != nil {
					primaryKey, isZero := valueOf(context.Background(), elemValue)
					if isZero {
						allZero = true
						break
					}
					allZero = false
					keyParts = append(keyParts, fmt.Sprintf("%v", primaryKey))
				}
			}
			if allZero {
				continue
			}
			primaryKeys = append(primaryKeys, strings.Join(keyParts, ":"))
		}
		objects = append(objects, elemValue.Interface())
	}
	return primaryKeys, objects
}

// getUniqueKeysFromObjects 从对象中提取unique键值
// 返回: map[uniqueIndexName]map[objectIndex]uniqueKey
func getUniqueKeysFromObjects(db *gorm.DB, objects []interface{}) map[string]map[int]string {
	if db.Statement.Schema == nil {
		return nil
	}

	uniqueIndexes := getAllUniqueIndexes(db.Statement.Schema)
	if len(uniqueIndexes) == 0 {
		return nil
	}

	result := make(map[string]map[int]string)
	for indexName, index := range uniqueIndexes {
		indexKeys := make(map[int]string)
		for objIdx, obj := range objects {
			objValue := reflect.Indirect(reflect.ValueOf(obj))
			if objValue.Kind() != reflect.Struct {
				continue
			}

			keyParts := make([]string, 0, len(index.Fields))
			allZero := true
			for _, fieldOption := range index.Fields {
				if fieldOption.Field == nil {
					allZero = true
					break
				}
				valueOf := fieldOption.Field.ValueOf
				if valueOf != nil {
					fieldValue, isZero := valueOf(context.Background(), objValue)
					if isZero {
						allZero = true
						break
					}
					allZero = false
					keyParts = append(keyParts, fmt.Sprintf("%v", fieldValue))
				}
			}
			if !allZero && len(keyParts) == len(index.Fields) {
				indexKeys[objIdx] = strings.Join(keyParts, ":")
			}
		}
		if len(indexKeys) > 0 {
			result[indexName] = indexKeys
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func uniqueStringSlice(slice []string) []string {
	retSlice := make([]string, 0)
	mmap := make(map[string]struct{})
	for _, str := range slice {
		_, ok := mmap[str]
		if !ok {
			mmap[str] = struct{}{}
			retSlice = append(retSlice, str)
		}
	}
	return retSlice
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
