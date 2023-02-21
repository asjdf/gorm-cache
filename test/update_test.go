package test

import (
	. "github.com/smartystreets/goconvey/convey"

	"github.com/asjdf/gorm-cache/cache"
	"gorm.io/gorm"
)

func testPrimaryUpdate(cache cache.Cache, db *gorm.DB) {
	err := cache.ResetCache()
	So(err, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 0)

	models := make([]*TestModel, 0)
	result := db.Where("id IN (?)", []int{1, 2, 3}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 0)

	models = make([]*TestModel, 0)
	result = db.Where("id IN (?)", []int{1, 2}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 1)
	So(len(models), ShouldEqual, 2)

	result = db.Model(models[0]).Where("id IN (1)").Updates(map[string]interface{}{"value8": -1})
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 1)

	models = make([]*TestModel, 0)
	result = db.Where("id IN (1,2)").Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 1)
	So(len(models), ShouldEqual, 2)
	So(models[0].Value8, ShouldEqual, -1)

	result = db.Table(TestModelTableName).Where("value8 = -1").UpdateColumn("value8", 1)
	So(result.Error, ShouldBeNil)

	models = make([]*TestModel, 0)
	result = db.Where("id IN (1,2)").Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 1)
}

func testSearchUpdate(cache cache.Cache, db *gorm.DB) {
	err := cache.ResetCache()
	So(err, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 0)

	models := make([]TestModel, 0)
	result := db.Where("id IN (?)", []int{51, 52}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 0)
	So(len(models), ShouldEqual, 2)

	result = db.Table(TestModelTableName).
		Where("id IN (?)", []int{53}).UpdateColumn("value5", 5)
	So(result.Error, ShouldBeNil)

	models = make([]TestModel, 0)
	result = db.Where("id IN (?)", []int{51, 52}).Find(&models)
	So(result.Error, ShouldBeNil)
	So(cache.HitCount(), ShouldEqual, 1)
	So(len(models), ShouldEqual, 2)
}
