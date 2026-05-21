// List-based config pool generator implementation
package domain

import "github.com/fulcrumproject/core/pkg/properties"

type ConfigPoolListGenerator = PoolListGenerator[*ConfigPoolValue]

func NewConfigPoolListGenerator(valueRepo ConfigPoolValueRepository, poolID properties.UUID) *ConfigPoolListGenerator {
	return NewPoolListGenerator(valueRepo, poolID)
}

var _ ConfigPoolGenerator = (*ConfigPoolListGenerator)(nil)
