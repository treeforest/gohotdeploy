// 参考代码：https://github.com/soupdiver/go-gitlab-webhook

package webhook

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
)

//GitlabRepository represents repository information from the webhook
type GitlabRepository struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Home        string `json:"home"`
}

//Commit represents commit information from the webhook
type Commit struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	URL       string `json:"url"`
	Author    Author `json:"author"`
}

//Author represents author information from the webhook
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

//Webhook represents push information from the webhook
type Webhook struct {
	Before            string           `json:"before"`
	After             string           `json:"after"`
	Ref               string           `json:"ref"`
	Username          string           `json:"username"`
	UserID            int              `json:"user_id"`
	ProjectID         int              `json:"project_id"`
	Repository        GitlabRepository `json:"repository"`
	Commits           []Commit         `json:"commits"`
	TotalCommitsCount int              `json:"total_commits_count"`
}

type DispatchFunc func(hook *Webhook)

func Handler(dispatch DispatchFunc) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		raw := r.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}
		log.Info().Msgf("IP:%s | METHOD:%s | PATH:%s", r.RemoteAddr, r.Method, path)

		var hook Webhook

		//read request body
		var data, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return
		}

		//unmarshal request body
		err = json.Unmarshal(data, &hook)
		if err != nil {
			return
		}

		//dispatch web hook
		dispatch(&hook)
	}
}
