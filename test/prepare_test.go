package test

import (
	"gorm.io/gorm"
	"strconv"
)

func PrepareTableAndData(db *gorm.DB) error {
	err := db.AutoMigrate(&TestModel{})
	if err != nil {
		return err
	}

	models := make([]TestModel, 0, testSize)
	for i := 1; i <= testSize; i++ {
		_pValue := int64(i)
		model := TestModel{
			ID:        int64(i),
			Value1:    int64(i),
			Value2:    int64(i),
			Value3:    int64(i),
			Value4:    int64(i),
			Value5:    int64(i),
			Value6:    int64(i),
			Value7:    int64(i),
			Value8:    int64(i),
			Value9:    strconv.Itoa(i),
			PtrValue1: &_pValue,
		}
		models = append(models, model)
	}

	return db.CreateInBatches(models, 2000).Error
}

func CleanTable(db *gorm.DB) error {
	return db.Migrator().DropTable(&TestModel{})
}
