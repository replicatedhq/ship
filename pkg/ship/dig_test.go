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
			name: "headless",
			set: map[string]bool{
				"headless":           true,
				"navigate-lifecycle": false,
			},
		},
		{
			name: "navigate",
			set: map[string]bool{
				"headless":           false,
				"navigate-lifecycle": true,
			},
		},
		{
			name: "headed",
			set: map[string]bool{
				"headless":           false,
				"navigate-lifecycle": false,
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

			container, err := buildInjector()
			req.NoError(err)

			err = container.Invoke(func(s *Ship) error {
				// don't do anything with it, just make sure we can get one
				return nil
			})

			req.NoError(err)

		})

	}
}
