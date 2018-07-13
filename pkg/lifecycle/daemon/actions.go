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
			Text:        "Skip",
			LoadingText: "Skipping",
			OnClick: ActionRequest{
				URI:    "/terraform/skip",
				Method: "POST",
			},
		},
	}
}
