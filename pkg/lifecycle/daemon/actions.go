package daemon

func MessageActions(id string) []Action {
	return []Action{
		{
			ButtonType:  "primary",
			Text:        "Confirm",
			LoadingText: "Confirming",
			OnClick: ActionRequest{
				URI:    "/message/confirm",
				Method: "POST",
				Body:   `{"step_id": "` + id + `"}`,
			},
		},
	}
}

func HelmIntroActions(id string) []Action {
	return []Action{
		{
			ButtonType:  "primary",
			Text:        "Get started",
			LoadingText: "Confirming",
			OnClick: ActionRequest{
				URI:    "/message/confirm",
				Method: "POST",
				Body:   `{"step_id": "` + id + `"}`,
			},
		},
	}
}

func HelmValuesActions() []Action {
	return []Action{
		{
			Sort:        0,
			ButtonType:  "primary",
			Text:        "Save values",
			LoadingText: "Saving",
			OnClick: ActionRequest{
				URI:    "/helm-values",
				Method: "POST",
				Body:   `{"step_name": "helm.values"}`,
			},
		},
		{
			Sort:        1,
			ButtonType:  "primary",
			Text:        "Continue",
			LoadingText: "Continuing",
			OnClick: ActionRequest{
				URI:    "/message/confirm",
				Method: "POST",
				Body:   `{"step_name": "helm.values"}`,
			},
		},
	}
}
