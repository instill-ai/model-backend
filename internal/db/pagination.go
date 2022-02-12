package db

import (
	"gorm.io/gorm"
)

type Pagination struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Sort   string `json:"sort"`
}

func (p *Pagination) GetOffset() int {
	return p.Offset
}

func (p *Pagination) GetLimit() int {
	return p.Limit
}

func (p *Pagination) GetSort() string {
	if p.Sort == "" {
		p.Sort = "Id asc"
	}
	return p.Sort
}

func Paginate(value interface{}, pagination *Pagination, db *gorm.DB) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(pagination.GetOffset()).Limit(pagination.GetLimit()).Order(pagination.GetSort())
	}
}
