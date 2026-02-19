package entity

import "errors"

var ErrUserNotFound = errors.New("user not found")
var ErrJobNotFound = errors.New("job not found")
var ErrUserAlreadyExists = errors.New("user already exists")
