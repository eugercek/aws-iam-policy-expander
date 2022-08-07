package fetch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	DOWNLOAD_URL  = "https://awspolicygen.s3.amazonaws.com/js/policies.js"
	REMOVE_PREFIX = "app.PolicyEditorConfig="
)

type PolicyDocument struct {
	ServiceMap map[string]Service `json:"serviceMap"`
}

type Service struct {
	StringPrefix string   `json:"StringPrefix"`
	Actions      []string `json:"Actions"`
}

func GetData() (data *PolicyDocument, err error) {
	fmt.Println("Downloading policies...")
	resp, err := http.Get(DOWNLOAD_URL)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	body = body[len(REMOVE_PREFIX):] // It's used for editor config

	err = json.Unmarshal(body, &data)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func ExpandAction(inp string, data *PolicyDocument) (ret []string, str string, err error) {
	args := strings.Split(inp, ":")

	if len(args) != 2 {
		return nil, "", errors.New("wrong type of input")
	}

	service := args[0]
	folded := args[1]

	if !strings.Contains(folded, "*") {
		return []string{folded}, "", nil
	}

	var actions []string

	for _, v := range data.ServiceMap {
		if v.StringPrefix == service {
			actions = v.Actions
			break
		}
	}

	// strings.Contains("foo", "") -> true
	s := strings.Replace(folded, "*", "", 1)

	// TODO Optimize
	for _, a := range actions {
		if strings.Contains(a, s) {
			ret = append(ret, service+":"+a)
		}
	}

	return ret, service, nil
}
