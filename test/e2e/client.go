package e2e

import "resty.dev/v3"

var baseUrl = "http://localhost:3000/api/v1"

func NewClient(authToken string) *resty.Client {
	return resty.New().SetBaseURL(baseUrl).SetAuthToken(authToken)
}
