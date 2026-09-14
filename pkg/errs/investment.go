package errs

import "net/http"

const NormalSubcategoryInvestment = 20

var (
	ErrInvestmentProtected = NewNormalError(NormalSubcategoryInvestment, 1, http.StatusConflict, "managed investment records must be changed through investment operations")
	ErrInvestmentConflict  = NewNormalError(NormalSubcategoryInvestment, 2, http.StatusConflict, "investment version or request key conflict; reload before retrying")
	ErrInvestmentInvalid   = NewNormalError(NormalSubcategoryInvestment, 3, http.StatusBadRequest, "invalid investment operation")
	ErrInvestmentNotFound  = NewNormalError(NormalSubcategoryInvestment, 4, http.StatusNotFound, "investment position not found")
)
