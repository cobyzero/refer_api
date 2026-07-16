package models

type UserSummaryUserDto struct {
	Name     string  `json:"name"`
	PhotoURL *string `json:"photoUrl"`
	Stars    int     `json:"stars"`
}

type ActivitySummaryDto struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Stars int    `json:"stars"`
	Date  string `json:"date"`
}

type UserSummaryResponse struct {
	User               UserSummaryUserDto   `json:"user"`
	Stars              int                  `json:"stars"`
	Rewards            []Reward             `json:"rewards"`
	RecentActivity     []ActivitySummaryDto `json:"recentActivity"`
	NextRewardProgress float64              `json:"nextRewardProgress"`
}

type AdminStatDto struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Change string `json:"change"`
	Icon   string `json:"icon"`
	Color  string `json:"color"`
}

type AdminActivitySummary struct {
	User   string `json:"user"`
	Action string `json:"action"`
	Time   string `json:"time"`
	Avatar string `json:"avatar"`
}

type AdminSummaryResponse struct {
	Stats          []AdminStatDto          `json:"stats"`
	RecentActivity []AdminActivitySummary  `json:"recentActivity"`
}
