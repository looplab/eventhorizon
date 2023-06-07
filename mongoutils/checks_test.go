package mongoutils

import (
	"testing"
)

func TestCheckValidCollectionName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"empty name",
			args{
				name: "",
			},
			true,
		},
		{
			"valid name",
			args{
				name: "valid",
			},
			false,
		},
		{
			"with spaces",
			args{
				name: "invalid name",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckCollectionName(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("CheckCollectionName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
