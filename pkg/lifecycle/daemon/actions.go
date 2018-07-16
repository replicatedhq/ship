package daemon

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
