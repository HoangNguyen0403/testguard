package domain

import (
	"testing"

	gm "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"example.com/app/internal/domain/mocks"
)

func TestOrderServiceUsesAliasedMockController(t *testing.T) {
	ctrl := gm.NewController(t)
	defer ctrl.Finish()
	stock := mocks.NewMockStockPort(ctrl)
	stock.EXPECT().Available("sku-2").Return(1, nil)

	svc := NewOrderService(stock)
	err := svc.Place(Order{SKU: "sku-2", Qty: 2})

	assert.NoError(t, err)
}
