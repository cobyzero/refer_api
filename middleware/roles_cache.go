package middleware

import (
	"sync"
	"time"
)

// Fase 0: in-memory TTL cache for app roles so API JWTs (which carry no
// roles) don't hit public.users on EVERY authenticated request.
// Single instance: sync.Map is enough. Redis (Fase 1) replaces this when we
// scale to >1 replica.
//
// TTL 5 min is a balance: role changes propagate with small delay, but the
// per-request FindByID+Roles disappears. Call InvalidateRolesCache on
// role updates/deletes to propagate immediately.

type rolesEntry struct {
	roles     []string
	expiresAt time.Time
}

var (
	rolesCache   sync.Map // map[string]rolesEntry
	rolesTTL     = 5 * time.Minute
	rolesInflight sync.Map // map[string]*call — singleflight-lite per userID
)

type call struct {
	wg    sync.WaitGroup
	roles []string
}

// getCachedRoles returns cached roles when fresh.
func getCachedRoles(userID string) ([]string, bool) {
	if v, ok := rolesCache.Load(userID); ok {
		if e, ok := v.(rolesEntry); ok {
			if time.Now().Before(e.expiresAt) {
				out := make([]string, len(e.roles))
				copy(out, e.roles)
				return out, true
			}
			rolesCache.Delete(userID)
		}
	}
	return nil, false
}

func setCachedRoles(userID string, roles []string) {
	out := make([]string, len(roles))
	copy(out, roles)
	rolesCache.Store(userID, rolesEntry{roles: out, expiresAt: time.Now().Add(rolesTTL)})
}

// InvalidateRolesCache drops the cached roles for a user.
// Call it after Create/Update/Delete user or role assignment changes.
func InvalidateRolesCache(userID string) {
	if userID == "" {
		return
	}
	rolesCache.Delete(userID)
}

// InvalidateAllRolesCache drops the whole roles cache (e.g. after bulk import).
func InvalidateAllRolesCache() {
	rolesCache.Range(func(k, _ interface{}) bool {
		rolesCache.Delete(k)
		return true
	})
}
