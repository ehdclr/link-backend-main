package req

type AdminCreateAdminRequest struct {
	Email       string       `json:"email" binding:"required,email"`
	Password    string       `json:"password" binding:"required"`
	Name        string       `json:"name" binding:"required"`
	Phone       string       `json:"phone" binding:"required"`
	Nickname    string       `json:"nickname" binding:"required"`
	UserProfile *UserProfile `json:"user_profile,omitempty"`
}

type AdminCreateCompanyRequest struct {
	CpName                    string `json:"cp_name" binding:"required"`
	CpNumber                  string `json:"cp_number,omitempty"`
	RepresentativeName        string `json:"representative_name,omitempty"`
	RepresentativePhoneNumber string `json:"representative_phone_number,omitempty"`
	RepresentativeEmail       string `json:"representative_email,omitempty"`
	RepresentativeAddress     string `json:"representative_address,omitempty"`
	RepresentativePostalCode  string `json:"representative_postal_code,omitempty"`
}
