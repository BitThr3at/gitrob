package core

import (
	"flag"
	"fmt"
)

type Options struct {
	CommitDepth       *int
	GithubAccessToken *string `json:"-"`
	NoExpandOrgs      *bool
	Threads           *int
	Save              *string `json:"-"`
	Load              *string `json:"-"`
	BindAddress       *string
	Port              *int
	Silent            *bool
	Debug             *bool
	Logins            []string
	RepoURL           *string // Single repository URL to scan
	ConfigPath        *string // Path to config.yaml file
}

func ParseOptions() (Options, error) {
	options := Options{
		CommitDepth:       flag.Int("commit-depth", 500, "Number of repository commits to process"),
		GithubAccessToken: flag.String("github-access-token", "", "GitHub access token to use for API requests"),
		NoExpandOrgs:      flag.Bool("no-expand-orgs", false, "Don't add members to targets when processing organizations"),
		Threads:           flag.Int("threads", 0, "Number of concurrent threads (default number of logical CPUs)"),
		Save:              flag.String("save", "", "Save session file"),
		Load:              flag.String("load", "", "Load session file"),
		BindAddress:       flag.String("bind-address", "127.0.0.1", "Address to bind web server to"),
		Port:              flag.Int("port", 9393, "Port to run web server on"),
		Silent:            flag.Bool("silent", false, "Suppress all output except for errors"),
		Debug:             flag.Bool("debug", false, "Print debugging information"),
		RepoURL:           flag.String("repo", "", "Single GitHub repository URL to scan (e.g. 'owner/repo')"),
		ConfigPath:        flag.String("config", "", "Path to config.yaml file (required)"),
	}

	flag.Parse()
	options.Logins = flag.Args()

	// Validate required config file path
	if *options.ConfigPath == "" {
		return options, fmt.Errorf("config file path is required. Use -config flag to specify the path to config.yaml")
	}

	return options, nil
}
