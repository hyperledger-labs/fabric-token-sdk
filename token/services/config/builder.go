package config

// Builder helps to build a TMS configuration
type Builder struct {
}

// NewBuilder returns a new builder with default settings
func NewBuilder() *Builder {
	return &Builder{}
}

// Build compiles the configuration in a stream of bytes that can be imported by the configuration service
func (b *Builder) Build() ([]byte, error) {
	return nil, nil
}
