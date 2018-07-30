package flags

import (
	"testing"

	"github.com/spf13/viper"
)

func Test_GetCurrentOrDeprecatedString(t *testing.T) {
	currentStringViperExample := viper.New()
	currentStringViperExample.Set("currentFlag", "123")

	deprecatedStringViperExample := viper.New()
	deprecatedStringViperExample.Set("deprecatedFlag", "456")

	currentAndDeprecatedStringViperExample := viper.New()
	currentAndDeprecatedStringViperExample.Set("currentFlag", "123")
	currentAndDeprecatedStringViperExample.Set("deprecatedFlag", "456")

	type args struct {
		v             *viper.Viper
		currentKey    string
		deprecatedKey string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Current",
			args: args{
				v:             currentStringViperExample,
				currentKey:    "currentFlag",
				deprecatedKey: "deprecatedFlag",
			},
			want: "123",
		},
		{
			name: "Deprecated",
			args: args{
				v:             deprecatedStringViperExample,
				currentKey:    "currentFlag",
				deprecatedKey: "deprecatedFlag",
			},
			want: "456",
		},
		{
			name: "Current and deprecated favors current",
			args: args{
				v:             currentAndDeprecatedStringViperExample,
				currentKey:    "currentFlag",
				deprecatedKey: "deprecatedFlag",
			},
			want: "123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCurrentOrDeprecatedString(tt.args.v, tt.args.currentKey, tt.args.deprecatedKey); got != tt.want {
				t.Errorf("getCurrentOrDeprecatedFlagString() = %v, want %v", got, tt.want)
			}
		})
	}
}
