package util

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
)

func GenInstanceId() string {
	charList := []byte("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	// 使用全局随机数生成器，Go 1.20+ 已自动初始化
	length := 5
	str := make([]byte, length)
	for i := 0; i < length; i++ {
		str[i] = charList[rand.Intn(len(charList))]
	}
	return string(str)
}

// GenPrimaryCacheKey 生成主键缓存key，支持单个主键和联合主键
// 如果传入多个参数，会按顺序用":"连接；如果只传入一个参数，直接使用（可能是已经连接好的联合主键）
func GenPrimaryCacheKey(instanceId string, tableName string, primaryKeyValues ...string) string {
	var key string
	if len(primaryKeyValues) == 1 {
		// 单个参数，直接使用（可能是单个主键值，也可能是已经连接好的联合主键）
		key = primaryKeyValues[0]
	} else {
		// 多个参数，用":"连接
		key = strings.Join(primaryKeyValues, ":")
	}
	return fmt.Sprintf("%s:%s:p:%s:%s", GormCachePrefix, instanceId, tableName, key)
}

// GenPrimaryCacheKeyFromMap 从map生成主键缓存key，支持联合主键
// primaryKeyMap: key为字段名，value为字段值，会自动按字段名排序以保证一致性
func GenPrimaryCacheKeyFromMap(instanceId string, tableName string, primaryKeyMap map[string]string) string {
	keys := make([]string, 0, len(primaryKeyMap))
	for k := range primaryKeyMap {
		keys = append(keys, k)
	}
	// 排序以保证key的一致性
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, primaryKeyMap[k])
	}
	return GenPrimaryCacheKey(instanceId, tableName, values...)
}

func GenPrimaryCachePrefix(instanceId string, tableName string) string {
	return GormCachePrefix + ":" + instanceId + ":p:" + tableName
}

// GenUniqueCacheKey 生成unique键缓存key，支持单个unique键和联合unique键
// 如果传入多个参数，会按顺序用":"连接；如果只传入一个参数，直接使用（可能是已经连接好的联合unique键）
func GenUniqueCacheKey(instanceId string, tableName string, uniqueIndexName string, uniqueKeyValues ...string) string {
	var key string
	if len(uniqueKeyValues) == 1 {
		// 单个参数，直接使用（可能是单个unique键值，也可能是已经连接好的联合unique键）
		key = uniqueKeyValues[0]
	} else {
		// 多个参数，用":"连接
		key = strings.Join(uniqueKeyValues, ":")
	}
	return fmt.Sprintf("%s:%s:u:%s:%s:%s", GormCachePrefix, instanceId, tableName, uniqueIndexName, key)
}

// GenUniqueCacheKeyFromMap 从map生成unique键缓存key，支持联合unique键
// uniqueKeyMap: key为字段名，value为字段值，会自动按字段名排序以保证一致性
func GenUniqueCacheKeyFromMap(instanceId string, tableName string, uniqueIndexName string, uniqueKeyMap map[string]string) string {
	keys := make([]string, 0, len(uniqueKeyMap))
	for k := range uniqueKeyMap {
		keys = append(keys, k)
	}
	// 排序以保证key的一致性
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, uniqueKeyMap[k])
	}
	return GenUniqueCacheKey(instanceId, tableName, uniqueIndexName, values...)
}

// GenUniqueCachePrefix 生成unique键缓存前缀
func GenUniqueCachePrefix(instanceId string, tableName string, uniqueIndexName string) string {
	return fmt.Sprintf("%s:%s:u:%s:%s", GormCachePrefix, instanceId, tableName, uniqueIndexName)
}

func GenSearchCacheKey(instanceId string, tableName string, sql string, vars ...interface{}) string {
	buf := strings.Builder{}
	buf.WriteString(sql)
	for _, v := range vars {
		pv := reflect.ValueOf(v)
		if pv.Kind() == reflect.Ptr {
			buf.WriteString(fmt.Sprintf(":%v", pv.Elem()))
		} else {
			buf.WriteString(fmt.Sprintf(":%v", v))
		}
	}
	return fmt.Sprintf("%s:%s:s:%s:%s", GormCachePrefix, instanceId, tableName, buf.String())
}

func GenSearchCachePrefix(instanceId string, tableName string) string {
	return GormCachePrefix + ":" + instanceId + ":s:" + tableName
}

func GenSingleFlightKey(tableName string, sql string, vars ...interface{}) string {
	buf := strings.Builder{}
	buf.WriteString(sql)
	for _, v := range vars {
		pv := reflect.ValueOf(v)
		if pv.Kind() == reflect.Ptr {
			buf.WriteString(fmt.Sprintf(":%v", pv.Elem()))
		} else {
			buf.WriteString(fmt.Sprintf(":%v", v))
		}
	}
	return fmt.Sprintf("%s:%s", tableName, buf.String())
}
