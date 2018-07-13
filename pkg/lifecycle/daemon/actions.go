package daemon

type ActionRequest struct {
	URI    string `json:"uri"`
	Method string `json:"method"`
	Body   string `json:"body"`
}

type Action struct {
	ButtonType  string        `json:"buttonType"`
	Text        string        `json:"text"`
	LoadingText string        `json:"loadingText"`
	OnClick     ActionRequest `json:"onclick"`
}

/*

	"buttonType":  "primary",
	"text":        "Confirm",
	"loadingText": "Confirming",
	"onclick": map[string]string{
		"uri":    "/message/confirm",
		"method": "POST",
		"body":   `{"step_name": "message"}`,
	},
*/
func MessageActions() []Action {
	return []Action{
		{
			ButtonType:  "primary",
			Text:        "Confirm",
			LoadingText: "Confirming",
			OnClick: ActionRequest{
				URI:    "/message/confirm",
				Method: "POST",
				Body:   `{"step_name": "message"}`,
			},
		},
	}
}

func TerraformActions() []Action {
	return []Action{
		{
			ButtonType:  "primary",
			Text:        "Apply",
			LoadingText: "Applying",
			OnClick: ActionRequest{
				URI:    "/terraform/apply",
				Method: "POST",
			},
		},
		{
			ButtonType:  "secondary-gray",
			Text:        "Apply",
			LoadingText: "Applying",
			OnClick: ActionRequest{
				URI:    "/terraform/skip",
				Method: "POST",
			},
		},
	}
}
