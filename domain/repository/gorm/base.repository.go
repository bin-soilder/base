package gorm

import(
	"context"
	"errors"
	_gorm "github.com/jinzhu/gorm"
	"github.com/xxxmicro/base/database/gorm"
	"github.com/xxxmicro/base/domain/repository"
	"github.com/xxxmicro/base/domain/model"
	breflect "github.com/xxxmicro/base/reflect"
)

type baseRepository struct {
	db *_gorm.DB
}

func NewBaseRepository(db *_gorm.DB) repository.BaseRepository {
	return &baseRepository{ db }
}

func (r *baseRepository) Create(c context.Context, m model.Model) error {
	db := gorm.SetSpanToGorm(c, r.db)

	return db.Create(m).Error
}

func (r *baseRepository) Update(c context.Context, m model.Model) error {
	db := gorm.SetSpanToGorm(c, r.db)

	return db.Save(m).Error
}

func (r *baseRepository) FindOne(c context.Context, condition interface{}, m model.Model) error {
	db := gorm.SetSpanToGorm(c, r.db)

	return db.Where(condition).Take(m).Error
}

func (r *baseRepository) Delete(c context.Context, m model.Model) error {
	// TODO 这里要做主键保护，如果 m 什么都没设置，这里将会删除表的所有记录
	return r.db.Delete(m).Error
}


func (r *baseRepository) Page(c context.Context, query *model.PageQuery, m model.Model) (page *model.Page, err error){
	items := breflect.MakeSlicePtr(m, 0, 0)

	ms := r.db.NewScope(m).GetModelStruct()

	dbHandler := r.db.Model(m)
	dbHandler, err = buildQuery(dbHandler, ms, query.Filters)
	if err != nil {
		return
	}

	dbHandler, err = buildSort(dbHandler, ms, query.Sort)
	if err != nil {
		return
	}

	total, pageCount, err := pageQuery(dbHandler, query.PageNo, query.PageSize, items)
	page = &model.Page{
		Total: total,
		PageNo: query.PageNo,
		PageSize: query.PageSize,
		PageCount: pageCount,
		Content: items,
	}
	
	return
}

func (r *baseRepository) Cursor(c context.Context, query *model.CursorQuery, m model.Model, resultPtr interface{}) (extra *model.CursorExtra, err error) {
	ms := r.db.NewScope(m).GetModelStruct()

	dbHandler := r.db.Model(m)
	dbHandler, err = buildQuery(dbHandler, ms, query.Filters)
	if err != nil {
		return
	}

	dbHandler, reverse, err := GormCursorFilter(dbHandler, ms, query)
	if err != nil {
		return
	}

	// items := breflect.MakeSlicePtr(m, 0, 0)

	if err = dbHandler.Limit(query.Size).Find(resultPtr).Error; err != nil {
		return
	}

	if reverse {
		breflect.SlicePtrReverse(resultPtr)
	}

	var minCursor interface{} = nil
	var maxCursor interface{} = nil

	count := breflect.SlicePtrLen(resultPtr)
	if count > 0 {
		minItem := breflect.SlicePtrIndexOf(resultPtr, 0)
		field, ok := FindColumn(query.CursorSort.Property, ms, dbHandler)
		if !ok {
			err = errors.New("field not found")
			return
		}
		
		minCursor, err = breflect.GetStructField(minItem, field.Name)
		if err != nil {
			return
		}

		maxItem := breflect.SlicePtrIndexOf(resultPtr, count-1)
		maxCursor, err = breflect.GetStructField(maxItem, field.Name)
		if err != nil {
			return
		}
	}

	extra = &model.CursorExtra{
		Direction: query.Direction,
		Size:      query.Size,
		HasMore:   count == query.Size,
		MinCursor: minCursor,
		MaxCursor: maxCursor,
	}

	return
}
