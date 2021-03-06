// Code generated by rest/model/codegen.go. DO NOT EDIT.

package model

import "github.com/mongodb/grip/send"

type APIGithubOptions struct {
	Account string `json:"account"`
	Repo    string `json:"repo"`
}

func APIGithubOptionsBuildFromService(t send.GithubOptions) *APIGithubOptions {
	m := APIGithubOptions{}
	m.Account = StringString(t.Account)
	m.Repo = StringString(t.Repo)
	return &m
}

func APIGithubOptionsToService(m APIGithubOptions) *send.GithubOptions {
	out := &send.GithubOptions{}
	out.Account = StringString(m.Account)
	out.Repo = StringString(m.Repo)
	return out
}
