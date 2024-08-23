package errors

import "errors"

var (
	// ErrEmailFieldMissing email field missing
	ErrEmailFieldMissing = errors.New("email field missing")
	// ErrEmailOrRoleFieldMissing email or User role field is missing
	ErrEmailOrRoleFieldMissing = errors.New("email or User role field is missing")
	// ErrNameOrEmailOrRoleFieldMissing name or email or User role field is missing
	ErrNameOrEmailOrRoleFieldMissing = errors.New("name or email or User role field is missing")
	// ErrEmailNotValid email is invalid
	ErrEmailNotValid = errors.New("email is invalid")
	// ErrPasswordMissingOrEmpty password is missing or empty
	ErrPasswordMissingOrEmpty = errors.New("password is missing or empty")
	// ErrPasswordTooLong password exceeds character length of 72 Bytes
	ErrPasswordTooLong = errors.New("password exceeds character length of 72 Bytes")
	// ErrServiceNameEmpty service name is missing or empty
	ErrServiceNameEmpty = errors.New("service name is empty")
	// ErrVersionTagEmpty version tag is missing or empty
	ErrVersionTagEmpty = errors.New("version tag is empty")
	// ErrInternal internal error
	ErrInternal = errors.New("internal error")
	// ErrFailureToProcessRequest failed to process request
	ErrFailureToProcessRequest = errors.New("failed to process request")
	// ErrInvalidEmailOrPass invalid email or password
	ErrInvalidEmailOrPass = errors.New("invalid email or password")
	// ErrInvalidUserID invalid user id
	ErrInvalidUserID = errors.New("invalid user id")
	// ErrUserWithSameEmailAlreadyExists user already exists with same email
	ErrUserWithSameEmailAlreadyExists = errors.New("user already exists with same email")
	// ErrUserDoesNotExist user doesn't exists
	ErrUserDoesNotExist = errors.New("user doesn't exists")
	// ErrUniqueKeyConstrainViolation duplicate key value violates unique constraint
	ErrUniqueKeyConstrainViolation = errors.New("duplicate key value violates unique constraint")
	// ErrInvalidServiceID invalid service id
	ErrInvalidServiceID = errors.New("invalid service id")
	// ErrServiceAlreadyExists service already exists
	ErrServiceAlreadyExists = errors.New("service already exists")
	// ErrServiceDoesNotExist service doesn't exist
	ErrServiceDoesNotExist = errors.New("service doesn't exist")
	// ErrServiceVersionAlreadyExists service version already exists
	ErrServiceVersionAlreadyExists = errors.New("service version already exists")
	// ErrServiceVersionDoesNotExist service version doesn't exist
	ErrServiceVersionDoesNotExist = errors.New("service version doesn't exist")
	// ErrAuthzHeaderMissing authorization header is missing
	ErrAuthzHeaderMissing = errors.New("authorization header is missing")
	// ErrMissingToken missing Token header
	ErrMissingToken = errors.New("invalid missing Bearer Token")
	// ErrMissingToken invalid Token header
	ErrInvalidToken = errors.New("invalid Token")
	// ErrTokenClaimMissing token lacks desired claims
	ErrTokenClaimMissing = errors.New("token lacks desired claims")
	// ErrInvalidOrExpiredToken invalid or expired JWT
	ErrInvalidOrExpiredToken = errors.New("invalid or expired JWT")
	// ErrUserNotAuthorized user not authorized
	ErrUserNotAuthorized = errors.New("user not authorized")
)
