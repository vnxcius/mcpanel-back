package handlers

type LoginRequest struct {
	Password string `json:"password"`
}

type RenewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RenewAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}
type Mod struct {
	Name string `json:"name"`
}

type ModList struct {
	Mods []Mod `json:"mods"`
}