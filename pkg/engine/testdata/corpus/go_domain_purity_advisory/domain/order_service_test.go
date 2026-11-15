package domain

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"example.com/app/internal/domain/mocks"
)

func TestOrderServiceRejectsWhenStockMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stock := mocks.NewMockStockPort(ctrl)
	stock.EXPECT().Available("sku-1").Return(0, nil)

	svc := NewOrderService(stock)
	err := svc.Place(Order{SKU: "sku-1", Qty: 2})

	assert.ErrorIs(t, err, ErrOutOfStock)
}
