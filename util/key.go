package util

import (
	"encoding/base64"
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

// joinKeyParts 对多段做 base64 编码后用 sep 连接，避免 "a:b","c" 与 "a","b:c" 碰撞
func joinKeyParts(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	encoded := make([]string, len(parts))
	for i, p := range parts {
		encoded[i] = base64.RawURLEncoding.EncodeToString([]byte(p))
	}
	return strings.Join(encoded, sep)
}

// GenPrimaryCacheKey 生成主键缓存key，支持单个主键和联合主键
// 联合主键各段会做 base64 编码，避免含 ":" 的值产生 key 碰撞
func GenPrimaryCacheKey(instanceId string, tableName string, primaryKeyValues ...string) string {
	key := joinKeyParts(primaryKeyValues, ":")
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
// 联合unique键各段会做 base64 编码，避免含 ":" 的值产生 key 碰撞
func GenUniqueCacheKey(instanceId string, tableName string, uniqueIndexName string, uniqueKeyValues ...string) string {
	key := joinKeyParts(uniqueKeyValues, ":")
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
