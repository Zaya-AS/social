package domain

import "errors"

var ErrUsernameTaken = errors.New("username already taken")

var ErrPostAlreadyExists = errors.New("post with this ID already exists")

var ErrUserNotFound = errors.New("user not found")

var ErrPostNotFound = errors.New("post not found")
