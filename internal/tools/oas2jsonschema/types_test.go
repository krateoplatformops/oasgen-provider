package oas2jsonschema

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultGeneratorConfig(t *testing.T) {
	t.Run("should return a config with default values", func(t *testing.T) {
		// Execute
		config := DefaultGeneratorConfig()

		// Assert
		assert.NotNil(t, config)
		assert.Equal(t, []string{"application/json"}, config.AcceptedMIMETypes)
		assert.Equal(t, []int{http.StatusOK, http.StatusCreated}, config.SuccessCodes)
	})
}
