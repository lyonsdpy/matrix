package repository

// Repositories holds all data-access instances.
// Add concrete repo fields here as the project grows.
type Repositories struct{}

func New() *Repositories {
	return &Repositories{}
}
