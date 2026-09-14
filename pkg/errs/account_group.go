package errs

import "net/http"

var (
	ErrAccountGroupInvalid  = NewNormalError(21, 1, http.StatusBadRequest, "invalid account group or root account")
	ErrAccountGroupNotFound = NewNormalError(21, 2, http.StatusNotFound, "account group not found")
	ErrAccountGroupConflict = NewNormalError(21, 3, http.StatusConflict, "account grouping changed or name already exists; refresh before retrying")
)
