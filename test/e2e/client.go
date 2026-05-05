//go:build e2e

package e2e

import "resty.dev/v3"

func NewClient(serverURL, authToken string) *resty.Client {
	return resty.New().
		SetBaseURL(serverURL + "/api/v1").
		SetAuthToken(authToken)
}
