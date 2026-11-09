package response

import api "github.com/poiuyttq/tripgo/internal/generated"

func domainProblem(status int32, code, title, detail, instance string) api.Problem {
	return api.Problem{
		Type:     "https://tripgo.example/problems/" + code,
		Title:    title,
		Status:   status,
		Code:     code,
		Detail:   &detail,
		Instance: &instance,
	}
}
