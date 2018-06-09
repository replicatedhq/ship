package ship

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// Make sure we can get an instance of ship
func TestDI(t *testing.T) {
	req := require.New(t)
	viper.Set("headless", true)
	viper.Set("customer-endpoint", "https://g.replicated.com")

	container, err := buildInjector()
	req.NoError(err)

	err = container.Invoke(func(s *Ship) error {
		// don't do anything with it, just make sure we can get one
		return nil
	})

	req.NoError(err)
}
