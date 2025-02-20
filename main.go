package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BitThr3at/gitrob/core"
)

var (
	sess *core.Session
	err  error
)

func GatherTargets(sess *core.Session) {
	sess.Stats.Status = core.StatusGathering
	sess.Out.Important("Gathering targets...\n")

	// Handle single repository scan
	if *sess.Options.RepoURL != "" {
		parts := strings.Split(*sess.Options.RepoURL, "/")
		if len(parts) != 2 {
			sess.Out.Error("Invalid repository format. Use 'owner/repo' format\n")
			os.Exit(1)
		}
		owner := parts[0]
		target, err := core.GetUserOrOrganization(owner, sess.GithubClient)
		if err != nil {
			sess.Out.Error(" Error retrieving information on %s: %s\n", owner, err)
			os.Exit(1)
		}
		sess.Out.Debug("%s (ID: %d) type: %s\n", *target.Login, *target.ID, *target.Type)
		sess.AddTarget(target)
		return
	}

	// Original user/org scanning logic
	for _, login := range sess.Options.Logins {
		target, err := core.GetUserOrOrganization(login, sess.GithubClient)
		if err != nil {
			sess.Out.Error(" Error retrieving information on %s: %s\n", login, err)
			continue
		}
		sess.Out.Debug("%s (ID: %d) type: %s\n", *target.Login, *target.ID, *target.Type)
		sess.AddTarget(target)
		if *sess.Options.NoExpandOrgs == false && *target.Type == "Organization" {
			sess.Out.Debug("Gathering members of %s (ID: %d)...\n", *target.Login, *target.ID)
			members, err := core.GetOrganizationMembers(target.Login, sess.GithubClient)
			if err != nil {
				sess.Out.Error(" Error retrieving members of %s: %s\n", *target.Login, err)
				continue
			}
			for _, member := range members {
				sess.Out.Debug("Adding organization member %s (ID: %d) to targets\n", *member.Login, *member.ID)
				sess.AddTarget(member)
			}
		}
	}
}

func GatherRepositories(sess *core.Session) {
	var ch = make(chan *core.GithubOwner, len(sess.Targets))
	var wg sync.WaitGroup
	var threadNum int

	// Handle single repository scan
	if *sess.Options.RepoURL != "" {
		parts := strings.Split(*sess.Options.RepoURL, "/")
		repo := parts[1]
		sess.Out.Important("Gathering repository: %s...\n", *sess.Options.RepoURL)
		repository, err := core.GetRepository(parts[0], repo, sess.GithubClient)
		if err != nil {
			sess.Out.Error(" Error retrieving repository %s: %s\n", *sess.Options.RepoURL, err)
			os.Exit(1)
		}
		sess.AddRepository(repository)
		return
	}

	// Original repository gathering logic
	if len(sess.Targets) == 1 {
		threadNum = 1
	} else if len(sess.Targets) <= *sess.Options.Threads {
		threadNum = len(sess.Targets) - 1
	} else {
		threadNum = *sess.Options.Threads
	}
	wg.Add(threadNum)
	sess.Out.Debug("Threads for repository gathering: %d\n", threadNum)
	for i := 0; i < threadNum; i++ {
		go func() {
			for {
				target, ok := <-ch
				if !ok {
					wg.Done()
					return
				}
				repos, err := core.GetRepositoriesFromOwner(target.Login, sess.GithubClient)
				if err != nil {
					sess.Out.Error(" Failed to retrieve repositories from %s: %s\n", *target.Login, err)
				}
				if len(repos) == 0 {
					continue
				}
				for _, repo := range repos {
					sess.Out.Debug(" Retrieved repository: %s\n", *repo.FullName)
					sess.AddRepository(repo)
				}
				sess.Stats.IncrementTargets()
				sess.Out.Info(" Retrieved %d %s from %s\n", len(repos), core.Pluralize(len(repos), "repository", "repositories"), *target.Login)
			}
		}()
	}

	for _, target := range sess.Targets {
		ch <- target
	}
	close(ch)
	wg.Wait()
}

func AnalyzeRepositories(sess *core.Session) {
	sess.Stats.Status = core.StatusAnalyzing
	var ch = make(chan *core.GithubRepository, len(sess.Repositories))
	var wg sync.WaitGroup
	var threadNum int
	if len(sess.Repositories) <= 1 {
		threadNum = 1
	} else if len(sess.Repositories) <= *sess.Options.Threads {
		threadNum = len(sess.Repositories) - 1
	} else {
		threadNum = *sess.Options.Threads
	}
	wg.Add(threadNum)
	sess.Out.Debug("Threads for repository analysis: %d\n", threadNum)

	sess.Out.Important("Analyzing %d %s...\n", len(sess.Repositories), core.Pluralize(len(sess.Repositories), "repository", "repositories"))

	for i := 0; i < threadNum; i++ {
		go func(tid int) {
			for {
				sess.Out.Debug("[THREAD #%d] Requesting new repository to analyze...\n", tid)
				repo, ok := <-ch
				if !ok {
					sess.Out.Debug("[THREAD #%d] No more tasks, marking WaitGroup as done\n", tid)
					wg.Done()
					return
				}

				sess.Out.Debug("[THREAD #%d][%s] Cloning repository...\n", tid, *repo.FullName)
				clone, path, err := core.CloneRepository(repo.CloneURL, repo.DefaultBranch, *sess.Options.CommitDepth)
				if err != nil {
					if err.Error() != "remote repository is empty" {
						sess.Out.Error("Error cloning repository %s: %s\n", *repo.FullName, err)
					}
					sess.Stats.IncrementRepositories()
					sess.Stats.UpdateProgress(sess.Stats.Repositories, len(sess.Repositories))
					continue
				}
				sess.Out.Debug("[THREAD #%d][%s] Cloned repository to: %s\n", tid, *repo.FullName, path)

				history, err := core.GetRepositoryHistory(clone)
				if err != nil {
					sess.Out.Error("[THREAD #%d][%s] Error getting commit history: %s\n", tid, *repo.FullName, err)
					os.RemoveAll(path)
					sess.Stats.IncrementRepositories()
					sess.Stats.UpdateProgress(sess.Stats.Repositories, len(sess.Repositories))
					continue
				}
				sess.Out.Debug("[THREAD #%d][%s] Number of commits: %d\n", tid, *repo.FullName, len(history))

				for _, commit := range history {
					sess.Out.Debug("[THREAD #%d][%s] Analyzing commit: %s\n", tid, *repo.FullName, commit.Hash)
					changes, _ := core.GetChanges(commit, clone)
					sess.Out.Debug("[THREAD #%d][%s] Changes in %s: %d\n", tid, *repo.FullName, commit.Hash, len(changes))
					for _, change := range changes {
						changeAction := core.GetChangeAction(change)
						path := core.GetChangePath(change)
						content, err := core.GetChangeContent(change)
						if err != nil {
							sess.Out.Debug("[THREAD #%d][%s] Error getting content for %s: %s\n", tid, *repo.FullName, path, err)
							continue
						}

						matchFile := core.NewMatchFile(path)
						if matchFile.IsSkippable() {
							sess.Out.Debug("[THREAD #%d][%s] Skipping %s\n", tid, *repo.FullName, matchFile.Path)
							continue
						}
						sess.Out.Debug("[THREAD #%d][%s] Matching: %s...\n", tid, *repo.FullName, matchFile.Path)

						// Create a temporary file with the content for content-based signatures
						if content != nil {
							tempDir, err := ioutil.TempDir("", "gitrob_content_")
							if err != nil {
								sess.Out.Debug("[THREAD #%d][%s] Error creating temp dir for %s: %s\n", tid, *repo.FullName, path, err)
								continue
							}
							defer os.RemoveAll(tempDir) // Clean up temp dir when done

							tempFile := filepath.Join(tempDir, filepath.Base(path))
							if err := ioutil.WriteFile(tempFile, content, 0644); err == nil {
								matchFile.Path = tempFile
							}
						}

						for _, signature := range core.Signatures {
							if signature.Match(matchFile) {
								finding := &core.Finding{
									FilePath:        path,
									Action:          changeAction,
									Description:     signature.Description(),
									Comment:         signature.Comment(),
									RepositoryOwner: *repo.Owner,
									RepositoryName:  *repo.Name,
									CommitHash:      commit.Hash.String(),
									CommitMessage:   strings.TrimSpace(commit.Message),
									CommitAuthor:    commit.Author.String(),
								}
								finding.Initialize()
								sess.AddFinding(finding)

								sess.Out.Warn(" %s: %s\n", strings.ToUpper(changeAction), finding.Description)
								sess.Out.Info("  Path.......: %s\n", finding.FilePath)
								sess.Out.Info("  Repo.......: %s\n", *repo.FullName)
								sess.Out.Info("  Message....: %s\n", core.TruncateString(finding.CommitMessage, 100))
								sess.Out.Info("  Author.....: %s\n", finding.CommitAuthor)
								if finding.Comment != "" {
									sess.Out.Info("  Comment....: %s\n", finding.Comment)
								}
								sess.Out.Info("  File URL...: %s\n", finding.FileUrl)
								sess.Out.Info("  Commit URL.: %s\n", finding.CommitUrl)
								sess.Out.Info(" ------------------------------------------------\n\n")
								sess.Stats.IncrementFindings()
								break
							}
						}
						sess.Stats.IncrementFiles()
					}
					sess.Stats.IncrementCommits()
					sess.Out.Debug("[THREAD #%d][%s] Done analyzing changes in %s\n", tid, *repo.FullName, commit.Hash)
				}
				sess.Out.Debug("[THREAD #%d][%s] Done analyzing commits\n", tid, *repo.FullName)
				os.RemoveAll(path)
				sess.Out.Debug("[THREAD #%d][%s] Deleted %s\n", tid, *repo.FullName, path)
				sess.Stats.IncrementRepositories()
				sess.Stats.UpdateProgress(sess.Stats.Repositories, len(sess.Repositories))
			}
		}(i)
	}
	for _, repo := range sess.Repositories {
		ch <- repo
	}
	close(ch)
	wg.Wait()
}

func PrintSessionStats(sess *core.Session) {
	sess.Out.Info("\nFindings....: %d\n", sess.Stats.Findings)
	sess.Out.Info("Files.......: %d\n", sess.Stats.Files)
	sess.Out.Info("Commits.....: %d\n", sess.Stats.Commits)
	sess.Out.Info("Repositories: %d\n", sess.Stats.Repositories)
	sess.Out.Info("Targets.....: %d\n\n", sess.Stats.Targets)
}

// Add this new function to handle repo list
func GatherTargetsFromRepoList(sess *core.Session) error {
	// Read repo list file
	content, err := ioutil.ReadFile(*sess.Options.RepoListFile)
	if err != nil {
		return fmt.Errorf("failed to read repo list file: %v", err)
	}

	// Split into lines and process each repo
	repos := strings.Split(string(content), "\n")
	for _, repoPath := range repos {
		// Skip empty lines
		repoPath = strings.TrimSpace(repoPath)
		if repoPath == "" {
			continue
		}

		// Split into owner/repo
		parts := strings.Split(repoPath, "/")
		if len(parts) != 2 {
			sess.Out.Error("Invalid repository format for %s. Skipping. Use 'owner/repo' format\n", repoPath)
			continue
		}

		owner := parts[0]
		repoName := parts[1]

		// Get the specific repository
		repo, err := core.GetRepository(owner, repoName, sess.GithubClient)
		if err != nil {
			sess.Out.Error(" Error retrieving repository %s: %s\n", repoPath, err)
			continue
		}
		sess.Out.Debug(" Retrieved repository: %s\n", *repo.FullName)
		sess.AddRepository(repo)
	}

	if len(sess.Repositories) == 0 {
		return fmt.Errorf("no valid repositories found in the list file")
	}
	return nil
}

func main() {
	if sess, err = core.NewSession(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	sess.Out.Info("%s\n\n", core.ASCIIBanner)
	sess.Out.Important("%s v%s started at %s\n", core.Name, core.Version, sess.Stats.StartedAt.Format(time.RFC3339))
	sess.Out.Important("Loaded %d signatures\n", len(core.Signatures))
	if !*sess.Options.NoWebServer {
		sess.Out.Important("Web interface available at http://%s:%d\n", *sess.Options.BindAddress, *sess.Options.Port)
	}

	if sess.Stats.Status == "finished" {
		sess.Out.Important("Loaded session file: %s\n", *sess.Options.Load)
	} else {
		// Check which mode we're running in
		if *sess.Options.RepoListFile != "" {
			// Process repositories from list file
			if err := GatherTargetsFromRepoList(sess); err != nil {
				sess.Out.Fatal("%v\n", err)
			}
			// Skip GatherRepositories since we already have the specific repos
			AnalyzeRepositories(sess)
			sess.Finish()
		} else if *sess.Options.RepoURL != "" {
			GatherTargets(sess)
			GatherRepositories(sess)
			AnalyzeRepositories(sess)
			sess.Finish()
		} else if len(sess.Options.Logins) > 0 {
			GatherTargets(sess)
			GatherRepositories(sess)
			AnalyzeRepositories(sess)
			sess.Finish()
		} else {
			sess.Out.Fatal("Please provide either a repository with -repo flag, a repo list file with -repo-list, or at least one GitHub organization/user\n")
		}

		if *sess.Options.Save != "" {
			err := sess.SaveToFile(*sess.Options.Save)
			if err != nil {
				sess.Out.Error("Error saving session to %s: %s\n", *sess.Options.Save, err)
			}
			sess.Out.Important("Saved session to: %s\n\n", *sess.Options.Save)
		}
	}

	PrintSessionStats(sess)

	if *sess.Options.NoWebServer {
		// Exit immediately if web server is disabled
		os.Exit(0)
	} else {
		sess.Out.Important("Press Ctrl+C to stop web server and exit.\n\n")
		select {}
	}
}
