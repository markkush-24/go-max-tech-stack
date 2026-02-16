package entity

type UserProfileDTO struct {
	ID      int64      `json:"id"`
	Name    string     `json:"name"`
	Email   string     `json:"email"`
	Profile ProfileDTO `json:"profile"`
}
type ProfileDTO struct {
	Bio  string `json:"bio"`
	City string `json:"city"`
}
