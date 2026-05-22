package registry

type PermissionScope struct {
	Type    string   `json:"type" binding:"required,oneof=namespace repository"`
	Class   string   `json:"class"`
	Name    string   `json:"name" binding:"required"`
	Actions []string `json:"actions" binding:"required"`
}

type NamespacePermissionItem struct {
	UserID   int      `json:"user_id" binding:"required"`
	UserName string   `json:"user_name,omitempty"`
	Actions  []string `json:"actions" binding:"required"`
}
