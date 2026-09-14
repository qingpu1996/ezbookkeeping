package errs

import "net/http"

var (
	ErrFundingPlanInvalid  = NewNormalError(22, 1, http.StatusBadRequest, "invalid funding plan; select owned monetary accounts in the planning currency")
	ErrFundingPlanConflict = NewNormalError(22, 2, http.StatusConflict, "funding plan changed on another device; reload before saving")
)
