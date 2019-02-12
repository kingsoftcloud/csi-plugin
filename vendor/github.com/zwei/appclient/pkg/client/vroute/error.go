package vroute

import "fmt"

type ErrorResponse struct {
	Detail  string `json:"detail"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// An Error represents a custom error for Aliyun API failure response
type Error struct {
	Response
	StatusCode    int           //Status Code of HTTP Response
	ErrorResponse ErrorResponse `json:"NeutronError"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("Kscyun API Error: RequestId: %s Status Code: %d Type: %s Message: %s Detail: %s", e.RequestId, e.StatusCode, e.ErrorResponse.Type, e.ErrorResponse.Message, e.ErrorResponse.Detail)
}

func GetClientErrorFromString(str string, req string) error {
	errors := ErrorResponse{
		Type:    "appclientFailure",
		Message: str,
	}
	rErrors := Error{
		ErrorResponse: errors,
		StatusCode:    -1,
	}
	rErrors.RequestId = req
	return &rErrors
}

func GetClientError(err error, req string) error {
	return GetClientErrorFromString(err.Error(), req)
}

