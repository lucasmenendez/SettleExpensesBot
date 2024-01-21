package bot

type Auth interface {
	AddAllowedUser(userID int64, alias string) error
	RemoveAllowedUser(userID int64) bool
	IsAllowed(userID int64) bool
	IsAdmin(userID int64) bool
	ListAllowedUsers() map[int64]string
	ListAdmins() map[int64]string
}
