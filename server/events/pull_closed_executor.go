// Copyright 2017 HootSuite Media Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Modified hereafter by contributors to runatlantis/atlantis.
//
package events

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/runatlantis/atlantis/server/events/locking"
	"github.com/runatlantis/atlantis/server/events/models"
	"github.com/runatlantis/atlantis/server/events/vcs"
)

//go:generate pegomock generate -m --use-experimental-model-gen --package mocks -o mocks/mock_pull_cleaner.go PullCleaner

// PullCleaner cleans up pull requests after they're closed/merged.
type PullCleaner interface {
	// CleanUpPull deletes the workspaces used by the pull request on disk
	// and deletes any locks associated with this pull request for all workspaces.
	CleanUpPull(repo models.Repo, pull models.PullRequest, host vcs.Host) error
}

// PullClosedExecutor executes the tasks required to clean up a closed pull
// request.
type PullClosedExecutor struct {
	Locker    locking.Locker
	VCSClient vcs.ClientProxy
	Workspace AtlantisWorkspace
}

type templatedProject struct {
	Path       string
	Workspaces string
}

var pullClosedTemplate = template.Must(template.New("").Parse(
	"Locks and plans deleted for the projects and workspaces modified in this pull request:\n" +
		"{{ range . }}\n" +
		"- path: `{{ .Path }}` {{ .Workspaces }}{{ end }}"))

// CleanUpPull cleans up after a closed pull request.
func (p *PullClosedExecutor) CleanUpPull(repo models.Repo, pull models.PullRequest, host vcs.Host) error {
	if err := p.Workspace.Delete(repo, pull); err != nil {
		return errors.Wrap(err, "cleaning workspace")
	}

	// Finally, delete locks. We do this last because when someone
	// unlocks a project, right now we don't actually delete the plan
	// so we might have plans laying around but no locks.
	locks, err := p.Locker.UnlockByPull(repo.FullName, pull.Num)
	if err != nil {
		return errors.Wrap(err, "cleaning up locks")
	}

	// If there are no locks then there's no need to comment.
	if len(locks) == 0 {
		return nil
	}

	templateData := p.buildTemplateData(locks)
	var buf bytes.Buffer
	if err = pullClosedTemplate.Execute(&buf, templateData); err != nil {
		return errors.Wrap(err, "rendering template for comment")
	}
	return p.VCSClient.CreateComment(repo, pull.Num, buf.String(), host)
}

// buildTemplateData formats the lock data into a slice that can easily be
// templated for the VCS comment. We organize all the workspaces by their
// respective project paths so the comment can look like:
// path: {path}, workspaces: {all-workspaces}
func (p *PullClosedExecutor) buildTemplateData(locks []models.ProjectLock) []templatedProject {
	workspacesByPath := make(map[string][]string)
	for _, l := range locks {
		path := l.Project.RepoFullName + "/" + l.Project.Path
		workspacesByPath[path] = append(workspacesByPath[path], l.Workspace)
	}

	// sort keys so we can write deterministic tests
	var sortedPaths []string
	for p := range workspacesByPath {
		sortedPaths = append(sortedPaths, p)
	}
	sort.Strings(sortedPaths)

	var projects []templatedProject
	for _, p := range sortedPaths {
		workspace := workspacesByPath[p]
		workspacesStr := fmt.Sprintf("`%s`", strings.Join(workspace, "`, `"))
		if len(workspace) == 1 {
			projects = append(projects, templatedProject{
				Path:       p,
				Workspaces: "workspace: " + workspacesStr,
			})
		} else {
			projects = append(projects, templatedProject{
				Path:       p,
				Workspaces: "workspaces: " + workspacesStr,
			})

		}
	}
	return projects
}
