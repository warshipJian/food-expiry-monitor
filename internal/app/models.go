package app

import "time"

type Food struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Barcode         string    `json:"barcode"`
	Category        string    `json:"category"`
	StorageLocation string    `json:"storageLocation"`
	Quantity        float64   `json:"quantity"`
	Unit            string    `json:"unit"`
	ExpiryDate      time.Time `json:"expiryDate"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type foodInput struct {
	Name            string  `json:"name" binding:"required,max=100"`
	Barcode         string  `json:"barcode" binding:"max=64"`
	Category        string  `json:"category" binding:"max=32"`
	StorageLocation string  `json:"storageLocation" binding:"max=32"`
	Quantity        float64 `json:"quantity" binding:"omitempty,gt=0"`
	Unit            string  `json:"unit" binding:"max=20"`
	ExpiryDate      string  `json:"expiryDate" binding:"required"`
}

type dashboard struct {
	Active   int `json:"active"`
	Expiring int `json:"expiring"`
	Expired  int `json:"expired"`
}
