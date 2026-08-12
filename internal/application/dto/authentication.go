package dto

import "time"

type LoginInput struct {
	Email    string
	Password string
}

type AuthenticationResult struct {
	AccessToken string
	ExpiresIn   time.Duration
}
