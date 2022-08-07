package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

func main() {
	single := flag.Bool("single", false, "convert single")
	file := flag.Bool("file", false, "expand inline in a file")

	flag.Parse()

	fmt.Println("Downloading policies...")
	data, err := getData()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *single {
		if flag.Args() == nil {
			fmt.Println("No action given")
		}

		actions, err := expandAction(flag.Args()[0], data)

		if err != nil {
			fmt.Println(err, flag.Args()[0])
		} else {
			for _, v := range actions {
				fmt.Println(v)
			}
		}
	} else if *file {
		if flag.Args() == nil {
			fmt.Println("No file given")
			os.Exit(1)
		}

		file, err := os.Open(flag.Args()[0])

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		bytes, _ := ioutil.ReadAll(file)

		var policy Policy

		err = json.Unmarshal(bytes, &policy)

		if err != nil {
			fmt.Println("can't parse policy", err)
			os.Exit(1)
		}

		sts := policy.Statements

		for i, st := range policy.Statements {
			var actions []string
			var setter func(int, []string)
			var elems []string

			if st.Action != nil {
				setter = func(i int, as []string) {
					sts[i].Action = as
				}
				elems = st.Action
			} else if st.NotAction != nil {
				setter = func(i int, nas []string) {
					sts[i].NotAction = nas
				}
				elems = st.NotAction
			} else {
				log.Fatal("Action or NotAction must be given.")
			}

			for _, str := range elems {
				if strings.Contains(str, "*") {
					exps, _ := expandAction(str, data)
					actions = append(actions, exps...)
				} else {
					actions = append(actions, str)
				}
				setter(i, actions)
				fmt.Println(actions)
			}
		}

		file.Close()

		f, _ := json.MarshalIndent(policy, "", "\t")
		err = ioutil.WriteFile("res.json", f, 0644)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		for {
			fmt.Print("Enter an AWS action:")
			var inp string
			fmt.Scanln(&inp)

			if inp == "exit" {
				break
			}

			actions, err := expandAction(inp, data)

			if err != nil {
				fmt.Println(err)
				continue
			}

			for _, v := range actions {
				fmt.Println(v)
			}
		}
	}
}

func getData() (data *PolicyDocument, err error) {
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

func expandAction(inp string, data *PolicyDocument) (ret []string, err error) {
	args := strings.Split(inp, ":")

	if len(args) != 2 {
		return nil, errors.New("wrong type of input")
	}

	service := args[0]
	folded := args[1]

	if !strings.Contains(folded, "*") {
		return []string{folded}, nil
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

	return ret, nil
}
