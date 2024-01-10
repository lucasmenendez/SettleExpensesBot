package bot

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Auth struct {
	admins       map[int64]string
	allowedUsers sync.Map
}

func InitAuth() (*Auth, error) {
	// parse admin users
	adminUsersIDs, err := parseIDs(os.Getenv("ADMIN_USER_IDS"))
	if err != nil {
		return nil, fmt.Errorf("invalid admin user ids: %w", err)
	}
	// parse admin users
	adminUsersAliases := parseStrs(os.Getenv("ADMIN_USER_ALIASES"))
	if len(adminUsersAliases) == 0 {
		return nil, fmt.Errorf("no admin user aliases provided")
	} else if len(adminUsersIDs) != len(adminUsersAliases) {
		return nil, fmt.Errorf("admin user ids and aliases must have the same length")
	}
	auth := &Auth{
		admins:       make(map[int64]string),
		allowedUsers: sync.Map{},
	}
	for i, id := range adminUsersIDs {
		auth.admins[id] = adminUsersAliases[i]
		auth.allowedUsers.Store(id, adminUsersAliases[i])
	}
	return auth, nil
}

func (a *Auth) AddAllowedUser(userID int64, alias string) error {
	if _, exists := a.allowedUsers.Load(userID); exists {
		return fmt.Errorf("user %d already added", userID)
	}
	a.allowedUsers.Store(userID, alias)
	return nil
}

func (a *Auth) RemoveAllowedUser(userID int64) bool {
	if _, exists := a.allowedUsers.Load(userID); !exists {
		return false
	}
	a.allowedUsers.Delete(userID)
	return true
}

func (a *Auth) ListAllowedUsers() map[int64]string {
	users := map[int64]string{}
	a.allowedUsers.Range(func(iUserID, iAlias interface{}) bool {
		userID, ok := iUserID.(int64)
		if !ok {
			return false
		}
		sAlias, ok := iAlias.(string)
		if !ok {
			return false
		}
		users[userID] = sAlias
		return true
	})
	return users
}

func (a *Auth) IsAdmin(userID int64) bool {
	_, ok := a.admins[userID]
	return ok
}

func (a *Auth) IsAllowed(userID int64) bool {
	_, ok := a.allowedUsers.Load(userID)
	return ok
}

func parseStrs(strs string) []string {
	return strings.Split(strings.TrimSpace(strs), ",")
}

func parseIDs(ids string) ([]int64, error) {
	var parsedIDs []int64
	for _, strID := range parseStrs(ids) {
		intID, err := strconv.ParseInt(strID, 10, 64)
		if err != nil {
			return nil, err
		}
		parsedIDs = append(parsedIDs, intID)
	}
	return parsedIDs, nil
}
