package entity

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age,omitempty"`
	Email string `json:"email"`
}
type UserDTO struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
type CreateUserInput struct {
	Name  string `json:"name"  validate:"required"`
	Age   int    `json:"age,omitempty" validate:"gte=0,lte=120"`
	Email string `json:"email" validate:"required,email"`
}
