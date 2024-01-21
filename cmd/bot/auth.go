package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type Auth struct {
	admins       map[int64]string
	allowedUsers sync.Map
}

func InitAuth(admins map[int64]string) *Auth {
	auth := &Auth{
		admins:       admins,
		allowedUsers: sync.Map{},
	}
	for id, alias := range admins {
		auth.allowedUsers.Store(id, alias)
	}
	return auth
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

func (a *Auth) ListAdmins() map[int64]string {
	return a.admins
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
