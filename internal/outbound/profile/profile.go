package profile

type Profile struct {
	UserID int64  `json:"user_id"`
	Bio    string `json:"bio"`
	City   string `json:"city"`
}
