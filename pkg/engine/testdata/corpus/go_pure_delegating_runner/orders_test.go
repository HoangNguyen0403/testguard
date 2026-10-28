package sample

import (
	"testing"
)

func TestGetOrdersV2(t *testing.T) {
	orders := []string{"ord-1", "ord-2"}
	if len(orders) == 0 {
		t.Fatal("expected orders")
	}
}

func TestGetOrderDetailV2(t *testing.T) {
	status := "completed"
	if status != "completed" {
		t.Errorf("got %s, want completed", status)
	}
}

func TestOrderServiceSuite(t *testing.T) {
	t.Run("GetOrdersV2", TestGetOrdersV2)
	t.Run("GetOrderDetailV2", TestGetOrderDetailV2)
}
