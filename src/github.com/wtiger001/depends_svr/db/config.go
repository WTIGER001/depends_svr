package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Config - Configuration Object
type JiraConfig struct {
	Filename             string   `json:"-"`
	User                 string   `json:"user"`
	Password             string   `json:"password"`
	JiraURL              string   `json:"jira_url"`
	Projects             []string `json:"projects"`
	CapabilityIssueType  string   `json:"capablity-issue-type"`
	FeatureIssueType     string   `json:"feature-issue-type"`
	RequirementIssueType string   `json:"requirement-issue-type"`
	ThreadIssueType      string   `json:"thread-issue-type"`
	ParentLink           string   `json:"parent-link"`
	ChildLink            string   `json:"child-link"`
	TracesToLink         string   `json:"traces-to-link"`
	TracesFromLink       string   `json:"traces-from-link"`
	DependsLinkOut       string   `json:"depends-link-out"`
	DependsLinkIn        string   `json:"depends-link-in"`
	ProcessPrefix        string   `json:"process-prefix"`
	Debug                bool     `json:"debug"`
	OutputFile           string   `json:"output-file"`
}

func (cfg *JiraConfig) Print() {
	fmt.Printf("%+v\n", cfg)
}

func (cfg *JiraConfig) Valid() bool {
	return !(cfg.User == "" || cfg.Password == "" || cfg.JiraURL == "")
}

func (cfg *JiraConfig) ApplyDefaults() {
	if cfg.Projects == nil {
		cfg.Projects = make([]string, 1, 1)
		cfg.Projects[0] = "PIR"
	}
	if cfg.CapabilityIssueType == "" {
		cfg.CapabilityIssueType = "New Capability"
	}
	if cfg.FeatureIssueType == "" {
		cfg.FeatureIssueType = "New Feature"
	}
	if cfg.RequirementIssueType == "" {
		cfg.RequirementIssueType = "Requirement"
	}
	if cfg.ThreadIssueType == "" {
		cfg.ThreadIssueType = "Thread"
	}
	if cfg.ParentLink == "" {
		cfg.ParentLink = "is parent of"
	}
	if cfg.ChildLink == "" {
		cfg.ChildLink = "is a child of"
	}
	if cfg.TracesToLink == "" {
		cfg.TracesToLink = "traces to"
	}
	if cfg.TracesFromLink == "" {
		cfg.TracesFromLink = "traces from"
	}
	if cfg.DependsLinkOut == "" {
		cfg.DependsLinkOut = "depends on"
	}
	if cfg.DependsLinkIn == "" {
		cfg.DependsLinkIn = "is a dependency of"
	}
	if cfg.ProcessPrefix == "" {
		cfg.ProcessPrefix = "process_"
	}
	if cfg.JiraURL == "" {
		cfg.JiraURL = "https://jira.di2e.net"
	}
	if cfg.OutputFile == "" {
		cfg.OutputFile = "output.json"
	}
	cfg.Debug = false
	cfg.User = "john.a.bauer"
	cfg.Password = "!QAZ@WSX#EDC4rfv5tgb"
}

// Load from file
func (cfg *JiraConfig) Load(filename string) (err error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var c JiraConfig
	json.Unmarshal(raw, &c)

	cfg.Filename = c.Filename
	cfg.User = c.User
	cfg.Password = c.Password
	cfg.JiraURL = c.JiraURL
	cfg.Projects = c.Projects
	cfg.CapabilityIssueType = c.CapabilityIssueType
	cfg.FeatureIssueType = c.FeatureIssueType
	cfg.RequirementIssueType = c.RequirementIssueType
	cfg.ThreadIssueType = c.ThreadIssueType
	cfg.ParentLink = c.ParentLink
	cfg.ChildLink = c.ChildLink
	cfg.TracesToLink = c.TracesToLink
	cfg.TracesFromLink = c.TracesFromLink
	cfg.ProcessPrefix = c.ProcessPrefix
	cfg.Debug = c.Debug

	return nil
}
