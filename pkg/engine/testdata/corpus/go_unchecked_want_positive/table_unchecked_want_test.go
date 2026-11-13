package sample_test

import "testing"

func TestTableUncheckedWant(t *testing.T) {
	tests := []struct {
		name    string
		in      int
		want    int
		wantErr bool
	}{
		{name: "ok", in: 1, want: 2, wantErr: false},
		{name: "fail", in: 2, want: 4, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in * 2; got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
