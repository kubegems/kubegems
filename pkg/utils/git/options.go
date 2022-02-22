package git

type Commiter struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type Options struct {
	Host      string    `json:"host" description:"git host"`
	Username  string    `json:"username" description:"git username"`
	Password  string    `json:"password" description:"git password"`
	Committer *Commiter `json:"committer" description:"git committer"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Host:     "http://gems-gitea:3000",
		Username: "root",
		Password: "",
		Committer: &Commiter{
			Name:  "service",
			Email: "service@example.com",
		},
	}
}
