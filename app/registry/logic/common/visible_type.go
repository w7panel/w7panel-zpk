package common

// 公共资源与命名空间沿用以下数据库值；资源继承值 3 不用于命名空间自身。
const (
	VisibleTypePrivate         = 1 // 私有：不公开任何操作，仅允许用户授权。
	VisibleTypePublicWrite     = 2 // 公开读写：匿名允许 pull 和 push。
	VisibleTypeFollowNamespace = 3 // 跟随命名空间的公开权限。
	VisibleTypePublicRead      = 4 // 公开只读：匿名仅允许 pull。
)

func PublicActionsForVisibleType(visibleType int) []string {
	switch visibleType {
	case VisibleTypePublicRead:
		return []string{string(PermissionActionTypePull)}
	case VisibleTypePublicWrite:
		return []string{string(PermissionActionTypePush), string(PermissionActionTypePull)}
	default:
		return nil
	}
}
