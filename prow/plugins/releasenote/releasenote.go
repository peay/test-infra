/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package releasenote

import (
	"fmt"
	"regexp"

	"github.com/Sirupsen/logrus"

	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/plugins"
)

const pluginName = "release-note"

const (
	releaseNoteActionRequired = "release-note-action-required"
	releaseNoteNone           = "release-note-none"
	releaseNoteLabelNeeded    = "release-note-label-needed"
	releaseNote               = "release-note"
)

var (
	allRNLabels = []string{
		releaseNoteNone,
		releaseNoteActionRequired,
		releaseNoteLabelNeeded,
		releaseNote,
	}

	releaseNoteRe     = regexp.MustCompile(`(?mi)^\/release-note\r?$`)
	releaseNoteNoneRe = regexp.MustCompile(`(?mi)^\/release-note-none\r?$`)
)

func init() {
	plugins.RegisterIssueCommentHandler(pluginName, handleIssueComment)
}

type githubClient interface {
	CreateComment(owner, repo string, number int, comment string) error
	AddLabel(owner, repo string, number int, label string) error
	RemoveLabel(owner, repo string, number int, label string) error
}

func handleIssueComment(pc plugins.PluginClient, ic github.IssueCommentEvent) error {
	return handle(pc.GitHubClient, pc.Logger, ic)
}

func handle(gc githubClient, log *logrus.Entry, ic github.IssueCommentEvent) error {
	// Only consider PRs and new comments.
	if !ic.Issue.IsPullRequest() || ic.Action != "created" {
		return nil
	}

	org := ic.Repo.Owner.Login
	repo := ic.Repo.Name
	number := ic.Issue.Number

	// Which label does the comment want us to add?
	var nl string
	if releaseNoteRe.MatchString(ic.Comment.Body) {
		nl = releaseNote
	} else if releaseNoteNoneRe.MatchString(ic.Comment.Body) {
		nl = releaseNoteNone
	} else {
		return nil
	}

	// Only allow authors and assignees to add labels.
	isAssignee := ic.Issue.IsAssignee(ic.Comment.User.Login)
	isAuthor := ic.Issue.IsAuthor(ic.Comment.User.Login)
	if !isAssignee && !isAuthor {
		resp := "you can only set release notes if you are the author or an assignee"
		log.Infof("Commenting with \"%s\".", resp)
		return gc.CreateComment(org, repo, number, plugins.FormatResponse(ic.Comment, resp))
	}

	// Add the requested label if necessary.
	var errs []error
	if !ic.Issue.HasLabel(nl) {
		log.Infof("Adding %s label.", nl)
		if err := gc.AddLabel(org, repo, number, nl); err != nil {
			errs = append(errs, err)
		}
	}
	// Remove all other release-note-* labels if necessary.
	for _, l := range allRNLabels {
		if l != nl && ic.Issue.HasLabel(l) {
			log.Infof("Removing %s label.", l)
			if err := gc.RemoveLabel(org, repo, number, l); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors setting labels: %v", len(errs), errs)
	}
	return nil
}
