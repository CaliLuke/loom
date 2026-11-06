package execution

// Workflow binds a stable durable workflow name to input and output types.
// Share this value between submitting processes and workers. The application
// owns the implementation and payload compatibility across deployments.
type Workflow[I, O any] struct {
	// Name identifies the workflow in the backend's database-wide namespace.
	Name string
}
