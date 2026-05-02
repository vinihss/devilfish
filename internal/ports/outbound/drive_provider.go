package outbound

// DriveFile represents a file in Google Drive.
// Deprecated: Use File from drive_capability.go instead.
type DriveFile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetDriveProvider returns the DriveCapability of an MCPClient if it supports it,
// along with a boolean indicating whether the capability is available.
func GetDriveProvider(client MCPClient) (DriveCapability, bool) {
	d, ok := client.(DriveCapability)
	return d, ok
}
