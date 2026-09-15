package utils

import (
	"github.com/gofiber/fiber/v2"
)

// Pagination holds normalized page/limit/offset values.
// Page is 1-based. Limit <= 0 means "no pagination" (legacy full list).
type Pagination struct {
	Page   int
	Limit  int
	Offset int
	Active bool
}

// ParsePagination reads ?page & ?limit (aliases pageSize/perPage) from the query.
// Backward-compatible: when neither is present it returns Active=false so
// handlers keep returning the legacy bare-array shape Flutter expects.
// Defaults when active: page=1, limit=defaultLimit. Clamped to [1, maxLimit].
func ParsePagination(c *fiber.Ctx, defaultLimit, maxLimit int) Pagination {
	page := c.QueryInt("page", 0)
	limit := c.QueryInt("limit", 0)
	if limit == 0 {
		limit = c.QueryInt("pageSize", 0)
	}
	if limit == 0 {
		limit = c.QueryInt("perPage", 0)
	}
	if page == 0 && limit == 0 {
		return Pagination{Active: false}
	}
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
		Active: true,
	}
}

// PagedEnvelope is the envelope returned ONLY when pagination is active.
// Legacy callers (no ?page/?limit) keep receiving the bare array.
func PagedEnvelope(data interface{}, total, page, limit int) fiber.Map {
	return fiber.Map{
		"data":  data,
		"total": total,
		"page":  page,
		"limit": limit,
	}
}
