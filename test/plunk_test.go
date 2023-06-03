package test

import (
	. "github.com/smartystreets/goconvey/convey"

	"github.com/asjdf/gorm-cache/cache"
	"gorm.io/gorm"
)

func testPluck(cache cache.Cache, db *gorm.DB) {
	var value9 []string
	result := db.Model(&TestModel{}).Pluck("value9", &value9)
	So(result.Error, ShouldBeNil)
	So(len(value9), ShouldEqual, testSize)
	So(value9[0], ShouldEqual, "1")
}
