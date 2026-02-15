package db

import "time"

// KeyValue is a generic key-value storage table for lightweight metadata.
type KeyValue struct {
	ID        uint   `gorm:"primaryKey"`
	Key       string `gorm:"size:100;uniqueIndex"`
	Value     string `gorm:"size:2000"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Member represents a merchant member/customer profile.
type Member struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:80;not null"`
	Phone     string `gorm:"size:20;uniqueIndex;not null"`
	Channel   string `gorm:"size:30;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Orders    []Order `gorm:"constraint:OnDelete:CASCADE"`
}

// Order represents a merchant order.
type Order struct {
	ID          uint       `gorm:"primaryKey"`
	OrderNo     string     `gorm:"size:40;uniqueIndex;not null"`
	MemberID    uint       `gorm:"index;not null"`
	AmountCents int64      `gorm:"not null"`
	Status      string     `gorm:"size:20;index;not null"`
	Source      string     `gorm:"size:30;not null"`
	PaidAt      *time.Time `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Member      Member `gorm:"foreignKey:MemberID"`
}
