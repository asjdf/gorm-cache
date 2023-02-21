package test

import (
	. "github.com/smartystreets/goconvey/convey"

	"github.com/asjdf/gorm-cache/cache"
	"gorm.io/gorm"
)

func testPrimaryDelete(cache cache.Cache, db *gorm.DB) {
	err := cache.ResetCache()
	So(err, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 0)

	models := make([]*TestModel, 0)
	result := db.Where("id IN (?)", []int{101, 102, 103, 104, 105}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 0)

	models = make([]*TestModel, 0)
	result = db.Where("id IN (?)", []int{101, 102, 103}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 1)
	So(len(models), ShouldEqual, 3)

	result = db.Delete(&TestModel{ID: 105})
	So(result.Error, ShouldBeNil)

	models = make([]*TestModel, 0)
	result = db.Where("id IN (?)", []int{101, 102, 103, 104}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 2)

	models = make([]*TestModel, 0)
	result = db.Where("id IN (?)", []int{101, 102, 103, 104, 105}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 2)

	result = db.Delete([]*TestModel{{ID: 103}, {ID: 104}})
	So(result.Error, ShouldBeNil)

	models = make([]*TestModel, 0)
	result = db.Where("id IN (?)", []int{101, 102}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 3)

	result = db.Where("id = 102").Delete(&TestModel{})
	So(result.Error, ShouldBeNil)

	result = db.Where("id IN (?)", []int{101}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 4)
}

func testSearchDelete(cache cache.Cache, db *gorm.DB) {
	err := cache.ResetCache()
	So(err, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 0)

	models := make([]TestModel, 0)
	result := db.Where("id IN (?)", []int{51, 52}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 0)
	So(len(models), ShouldEqual, 2)

	result = db.Delete(&TestModel{ID: 53})
	So(result.Error, ShouldBeNil)

	models = make([]TestModel, 0)
	result = db.Where("id IN (?)", []int{51, 52}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 0)
	So(len(models), ShouldEqual, 2)
}
