package util

import "testing"

func TestIsLegalPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "relative path",
			path:    "./happy/path",
			wantErr: false,
		},
		{
			name:    "absolute path",
			path:    "/unhappy/path",
			wantErr: true,
		},
		{
			name:    "relative parent path",
			path:    "../../unhappy/path",
			wantErr: true,
		},
		{
			name:    "embedded relative parent path",
			path:    "./happy/../../../unhappy/path",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IsLegalPath(tt.path); (err != nil) != tt.wantErr {
				t.Errorf("IsLegalPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
