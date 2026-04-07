package entity

type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Age     int    `json:"age,omitempty"`
	Email   string `json:"email"`
	Version int64  `json:"-"`
}
type UserDTO struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
type CreateUserInput struct {
	Name  string `json:"name"`
	Age   int    `json:"age,omitempty"`
	Email string `json:"email"`
}

type UserDTOV2 struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}
type CreateUserInputV2 struct {
	FullName string `json:"full_name"`
	Age      int    `json:"age,omitempty"`
	Email    string `json:"email"`
}

type UserExport struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func MapCreateV2ToV1(in CreateUserInputV2) CreateUserInput {
	return CreateUserInput{
		Name:  in.FullName,
		Age:   in.Age,
		Email: in.Email,
	}
}

func MapUserDTOToV2(u *UserDTO) UserDTOV2 {
	return UserDTOV2{
		ID:       u.ID,
		FullName: u.Name,
		Email:    u.Email,
	}
}

func MapUsersToV2(users []*User) []UserDTOV2 {
	out := make([]UserDTOV2, 0, len(users))
	for _, u := range users {
		out = append(out, UserDTOV2{
			ID:       u.ID,
			FullName: u.Name,
			Email:    u.Email,
		})
	}
	return out
}
