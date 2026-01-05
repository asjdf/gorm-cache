package util

import (
	"fmt"
	"math/rand"
	"reflect"
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

func GenPrimaryCacheKey(instanceId string, tableName string, primaryKey string) string {
	return fmt.Sprintf("%s:%s:p:%s:%s", GormCachePrefix, instanceId, tableName, primaryKey)
}

func GenPrimaryCachePrefix(instanceId string, tableName string) string {
	return GormCachePrefix + ":" + instanceId + ":p:" + tableName
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
