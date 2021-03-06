package ship

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// Make sure we can get an instance of ship
func TestDI(t *testing.T) {
	tests := []struct {
		name string
		set  map[string]bool
	}{
		{
			name: "headless+navcycle",
			set: map[string]bool{
				"headless": true,
				"navcycle": true,
			},
		},
		{
			name: "headless",
			set: map[string]bool{
				"headless": true,
				"navcycle": false,
			},
		},
		{
			name: "navcycle",
			set: map[string]bool{
				"headless": false,
				"navcycle": true,
			},
		},
		{
			name: "headed",
			set: map[string]bool{
				"headless": false,
				"navcycle": false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for key, value := range test.set {
				viper.Set(key, value)
				viper.Set("customer-endpoint", "https://g.replicated.com")
			}

			req := require.New(t)

			container, err := buildInjector(viper.GetViper())
			req.NoError(err)

			err = container.Invoke(func(s *Ship) error {
				// don't do anything with it, just make sure we can get one
				return nil
			})

			req.NoError(err)

		})

	}
}
