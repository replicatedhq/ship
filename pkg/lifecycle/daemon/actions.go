package daemon

import "github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"

func MessageActions() []daemontypes.Action {
	return []daemontypes.Action{
		{
			ButtonType:  "primary",
			Text:        "Confirm",
			LoadingText: "Confirming",
			OnClick: daemontypes.ActionRequest{
				URI:    "/message/confirm",
				Method: "POST",
				Body:   `{"step_name": "message"}`,
			},
		},
	}
}

func HelmIntroActions() []daemontypes.Action {
	return []daemontypes.Action{
		{
			ButtonType:  "primary",
			Text:        "Get started",
			LoadingText: "Confirming",
			OnClick: daemontypes.ActionRequest{
				URI:    "/message/confirm",
				Method: "POST",
				Body:   `{"step_name": "helm.intro"}`,
			},
		},
	}
}

func HelmValuesActions() []daemontypes.Action {
	return []daemontypes.Action{
		{
			Sort:        0,
			ButtonType:  "primary",
			Text:        "Save values",
			LoadingText: "Saving",
			OnClick: daemontypes.ActionRequest{
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
			OnClick: daemontypes.ActionRequest{
				URI:    "/message/confirm",
				Method: "POST",
				Body:   `{"step_name": "helm.values"}`,
			},
		},
	}
}
