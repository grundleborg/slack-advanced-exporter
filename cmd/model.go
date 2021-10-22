package cmd

type SlackFile struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	UrlPrivate         string `json:"url_private"`
	UrlPrivateDownload string `json:"url_private_download"`
}

type SlackPost struct {
	User    string       `json:"user"`
	Type    string       `json:"type"`
	Subtype string       `json:"subtype"`
	Text    string       `json:"text"`
	Ts      string       `json:"ts"`
	File    *SlackFile   `json:"file"`
	Files   []*SlackFile `json:"files"`
}

// As it appears in users.json and /api/users.list.
// There're obviously many more fields, but we only need a couple of them.
type SlackUser struct {
	Id      string           `json:"id"`
	Profile SlackUserProfile `json:"profile"`
}

type SlackUserProfile struct {
	Email string `json:"email"`
}
