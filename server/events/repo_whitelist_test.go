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
package events_test

import (
	"testing"

	"github.com/runatlantis/atlantis/server/events"
	. "github.com/runatlantis/atlantis/testing"
)

func TestIsWhitelisted(t *testing.T) {
	cases := []struct {
		Description  string
		Whitelist    string
		RepoFullName string
		Hostname     string
		Exp          bool
	}{
		{
			"exact match",
			"github.com/owner/repo",
			"owner/repo",
			"github.com",
			true,
		},
		{
			"exact match shouldn't match anything else",
			"github.com/owner/repo",
			"owner/rep",
			"github.com",
			false,
		},
		{
			"* should match anything",
			"*",
			"owner/repo",
			"github.com",
			true,
		},
		{
			"github.com* should match anything github",
			"github.com*",
			"owner/repo",
			"github.com",
			true,
		},
		{
			"github.com* should not match gitlab",
			"github.com*",
			"owner/repo",
			"gitlab.com",
			false,
		},
		{
			"github.com/o* should match",
			"github.com/o*",
			"owner/repo",
			"github.com",
			true,
		},
		{
			"github.com/owner/rep* should not match",
			"github.com/owner/rep*",
			"owner/re",
			"github.com",
			false,
		},
		{
			"github.com/owner/rep* should match",
			"github.com/owner/rep*",
			"owner/rep",
			"github.com",
			true,
		},
		{
			"github.com/o* should not match",
			"github.com/o*",
			"somethingelse/repo",
			"github.com",
			false,
		},
		{
			"github.com/owner/repo* should match exactly",
			"github.com/owner/repo*",
			"owner/repo",
			"github.com",
			true,
		},
		{
			"github.com/owner/* should match anything in org",
			"github.com/owner/*",
			"owner/repo",
			"github.com",
			true,
		},
		{
			"github.com/owner/* should not match anything not in org",
			"github.com/owner/*",
			"otherorg/repo",
			"github.com",
			false,
		},
		{
			"if there's any * it should match",
			"github.com/owner/repo,*",
			"otherorg/repo",
			"github.com",
			true,
		},
		{
			"any exact match should match",
			"github.com/owner/repo,github.com/otherorg/repo",
			"otherorg/repo",
			"github.com",
			true,
		},
		{
			"longer shouldn't match on exact",
			"github.com/owner/repo",
			"owner/repo-longer",
			"github.com",
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.Description, func(t *testing.T) {
			w := events.RepoWhitelist{Whitelist: c.Whitelist}
			Equals(t, c.Exp, w.IsWhitelisted(c.RepoFullName, c.Hostname))
		})
	}
}
