package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/EconomistWhichMBA/go.github/webhooks"
	"os"
	"os/exec"
	"strings"
	"time"
)

var port string = "7777"
var branch string = "master"
var command string
var command_args []string
var verbose bool = false

var repo string
var releases string

var args []string

var keep_at_least_backups = 3
var keep_at_least_daily = 7
var keep_at_least_monthly = 1

func init() {
	flag.StringVar(&port, "p", port, "Listen on a particular port.")
	flag.StringVar(&branch, "b", branch, "Listen for a particular branch.")
	flag.StringVar(&command, "e", command, "Execute a command post deploy.")
	flag.BoolVar(&verbose, "v", verbose, "Verbose.")
}

func log(str string, args ...interface{}) {
	if verbose {
		fmt.Printf(str, args...)
	}
}

func listenForPayloads(payloads chan *webhooks.GitHubPayload,
	repo,
	branch,
	releases,
	command string,
	command_args []string) {
	repos := strings.Split(repo, "/")
	repo_org := repos[0]
	repo_name := repos[1]
	ref := "refs/heads/" + branch

	for {
		payload := <-payloads
		log("\nReceived:\n%+v\n", payload)

		if payload.Ref == ref && payload.Repository.Organization == repo_org && payload.Repository.Name == repo_name {
			if err := doRelease(payload.HeadCommit.Id); err != nil {
				log(err.Error())
				continue
			}

			// TODO: Build in archiving of old releases into archives. see: http://golang.org/pkg/archive/tar/#example_

			// Post deploy command.
			if command != "" {
				out, err := exec.Command(command, command_args...).Output()
				if err != nil {
					log("Failed to run command:\n%s %s\n", command, strings.Join(command_args, " "))
				}
				log(string(out))
			}
		} else {
			log("Deploy skipped: " + payload.Ref + "\n")
		}
	}
}

func doRelease(commit string) error {
	if err := os.Chdir("working"); err != nil {
		errors.New("Failed to chdir to working\n")
		return err
	}
	// Deploy.
	out, err := exec.Command("git", "pull", "origin", branch).Output()
	if err != nil {
		log("Failed to update working copy.")
		return err
	}
	log(string(out))

	if err := os.Chdir("../"); err != nil {
		log("Failed to chdir to working\n")
		return err
	}

	t := time.Now()
	releasedir := "releases/" + fmt.Sprintf("%d-%02d-%02d.%02d%02d%02d.%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), commit)
	log("Release to: " + releasedir + "\n")

	err = CopyDir("working", releasedir)
	if err != nil {
		return err
	}

	os.Remove("current")
	err = os.Symlink(releasedir, "current")
	if err != nil {
		return err
	}

	return nil
}

func prepareDirectories(releases, repo, branch string) error {
	// Does releases directory exist and is writable?
	if err := checkDir(releases); err != nil {
		return err
	}
	if err := os.Chdir(releases); err != nil {
		log("Failed to chdir to " + releases + "/working\n")
		return err
	}
	if _, err := os.Create("release-test"); err != nil {
		log("The releases directory is not writable")
		return err
	}
	os.Remove("release-test")

	// Does the releases sub-directory exist.
	if err := checkDir("releases"); err != nil {
		return err
	}

	// Does the archives sub-directory exist.
	if err := checkDir("archives"); err != nil {
		return err
	}

	// Does working directory exist?
	if fi, err := os.Stat("working"); err != nil || !fi.IsDir() {
		out, err := exec.Command("git", "clone", "git@github.com:"+repo+".git", "working").Output()
		if err != nil {
			log("Failed to clone repository: git@github.com:" + repo + ".git")
			return err
		}
		log("Cloning repository:\n")
		log(string(out))

		if err := os.Chdir("working"); err != nil {
			log("Failed to chdir to working\n")
			return err
		}

		out, err = exec.Command("git", "checkout", branch).Output()
		if err != nil {
			log("Failed to check out " + branch + " into working copy.")
			return err
		}
		log(string(out))

		if err := os.Chdir("../"); err != nil {
			log("Failed to chdir to " + releases + "\n")
			return err
		}
	}

	return nil
}

func checkDir(dir string) error {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			return errors.New("Failed to create directory: " + dir)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	args = flag.Args()

	if len(args) != 2 {
		log("Usage: gitdeploy org/repo release/directory\n")
		return
	}
	repo = args[0]
	releases = args[1]

	if strings.Count(repo, "/") != 1 {
		log("The first argument should be of the format: organization/repository\n")
		return
	}

	log("repo: %v\n", repo)
	log("releases: %v\n", releases)

	log("branch: %v\n", branch)
	log("port: %v\n", port)
	log("command: %v\n", command)

	if command != "" && len(command) > 0 {
		command_args = strings.Fields(command)
		command = command_args[0]
		command_args = command_args[1:]
	}

	err := prepareDirectories(releases, repo, branch)
	if err != nil {
		log(err.Error() + "\n")
		return
	}

	payloads := make(chan *webhooks.GitHubPayload)
	go listenForPayloads(payloads, repo, branch, releases, command, command_args)

	webhooks.WebhookListener(port, payloads)
}
