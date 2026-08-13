package errors

const (
	CodeInvalidRequest     = "INVALID_REQUEST"
	CodeValidation         = "VALIDATION_ERROR"
	CodeInvalidUserID      = "INVALID_USER_ID"
	CodeEmailAlreadyExists = "EMAIL_ALREADY_EXISTS"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeUserNotFound       = "USER_NOT_FOUND"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeInternal           = "INTERNAL_ERROR"
)

const (
	MessageInvalidRequest     = "invalid request body"
	MessageInvalidUserID      = "user ID is invalid"
	MessageEmailAlreadyExists = "email already exists"
	MessageInvalidCredentials = "invalid credentials"
	MessageUserNotFound       = "user not found"
	MessageUnauthorized       = "authentication is required"
	MessageInternal           = "internal server error"
)
