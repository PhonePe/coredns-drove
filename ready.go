package drovedns

// Checks if apps data could be synced from drove cluster
func (e *DroveHandler) Ready() bool {
	return e.DroveEndpoints.AppsDB != nil
}
