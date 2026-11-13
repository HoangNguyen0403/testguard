package sample_test

import "testing"

func TestTableCheckedWant(t *testing.T) {
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
			got, err := Calculate(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Calculate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Calculate() got = %v, want %v", got, tt.want)
			}
		})
	}
}
