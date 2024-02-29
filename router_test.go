package pathrouter

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

var kindNames = map[uint8]string{
	staticKind:   "static",
	paramKind:    "param",
	trailingKind: "*",
}

func printNode(n *node[int], level int) {
	fmt.Println(strings.Repeat("  ", level-1) + "{")
	fmt.Printf("%skind: %q\n", strings.Repeat("  ", level), kindNames[n.kind])
	fmt.Printf("%send: %t\n", strings.Repeat("  ", level), n.end)
	fmt.Printf("%swildChild: %t\n", strings.Repeat("  ", level), n.wildChild)
	fmt.Printf("%spath: %q\n", strings.Repeat("  ", level), n.path)
	fmt.Printf("%svalue: %v\n", strings.Repeat("  ", level), n.value)
	if len(n.children) != 0 {
		fmt.Printf("%sindices: %q\n", strings.Repeat("  ", level), n.indices)
		fmt.Printf("%schildren: [\n", strings.Repeat("  ", level))
		for _, nn := range n.children {
			printNode(nn, level+2)
		}
		fmt.Printf("%s]\n", strings.Repeat("  ", level))
	}
	fmt.Println(strings.Repeat("  ", level-1) + "}")
}

func printRouter(r *Router[int]) {
	fmt.Print("Router ")
	if r.root != nil {
		printNode(r.root, 1)
	} else {
		fmt.Println("nil")
	}
}

func TestParams_Get(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name  string
		ps    Params
		args  args
		want  string
		want1 bool
	}{
		{ps: Params{{Key: "a", Value: "1"}, {Key: "b", Value: "2"}, {Key: "b", Value: "3"}}, args: args{"b"}, want: "2", want1: true},
		{ps: Params{{Key: "a", Value: "1"}, {Key: "b", Value: "2"}, {Key: "b", Value: "3"}}, args: args{"c"}, want: "", want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.ps.Get(tt.args.key)
			if got != tt.want {
				t.Errorf("Params.Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Params.Get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_splitPathSegment(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		{args: args{""}, want: "", want1: ""},
		{args: args{"a"}, want: "a", want1: ""},
		{args: args{":a"}, want: ":a", want1: ""},
		{args: args{":"}, want: ":", want1: ""},
		{args: args{"*"}, want: "*", want1: ""},
		{args: args{"/a"}, want: "/a", want1: ""},
		{args: args{"a/:a"}, want: "a/", want1: ":a"},
		{args: args{"a/b:a"}, want: "a/b", want1: ":a"},
		{args: args{"a/*"}, want: "a/", want1: "*"},
		{args: args{":a/a"}, want: ":a", want1: "/a"},
		{args: args{"*/a"}, want: "*", want1: "/a"},
		{args: args{":a*"}, want: ":a", want1: "*"},
		{args: args{"*:a"}, want: "*", want1: ":a"},
		{args: args{":a:b"}, want: ":a", want1: ":b"},
		{args: args{"**"}, want: "*", want1: "*"},
		{args: args{":*"}, want: ":", want1: "*"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := splitPathSegment(tt.args.path)
			if got != tt.want {
				t.Errorf("splitPathSegment() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("splitPathSegment() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestRouter_Add(t *testing.T) {
	tests := []struct {
		name    string
		routes  []string
		wantErr error
	}{
		{name: "nil"},
		{name: "empty", routes: []string{""}},
		{name: "static", routes: []string{"a"}},
		{name: "same", routes: []string{"a/b", "a/c", "a/b"}},
		{name: "static4", routes: []string{"a", "aba", "abc", "ac"}},
		{name: "empty-root", routes: []string{"a", "b"}},
		{name: "param", routes: []string{":a"}},
		{name: "param1", routes: []string{"/:a"}},
		{name: "param2", routes: []string{"/a/:a", "/b/:b"}},
		{name: "trailing", routes: []string{"*"}},
		{name: "trailing2", routes: []string{"/a/*", "/b/*"}},
		{name: "bad1-0", routes: []string{"*:a"}, wantErr: ErrInvalidPath},
		{name: "bad1-1", routes: []string{":"}, wantErr: ErrInvalidPath},
		{name: "bad1-2", routes: []string{":a:b"}, wantErr: ErrInvalidPath},
		{name: "bad1-3", routes: []string{":a*"}, wantErr: ErrInvalidPath},
		{name: "bad2", routes: []string{"/a", "/a/*:a"}, wantErr: ErrInvalidPath},
		{name: "conflict", routes: []string{"/a", "/:b"}, wantErr: ErrConflict},
		{name: "conflict2", routes: []string{"/:a", "/:ab"}, wantErr: ErrConflict},
		{name: "conflict3", routes: []string{"/:ab", "/:ac"}, wantErr: ErrConflict},
		{name: "conflict4", routes: []string{"/a/b/:c", "/a/*"}, wantErr: ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorOccurred := false
			r := &Router[int]{}
			for i, route := range tt.routes {
				err := r.Add(route, 100+i)
				if err != nil {
					if err != tt.wantErr {
						t.Fatalf("Router.Add() error = %v, wantErr %v", err, tt.wantErr)
					}
					errorOccurred = true
				}
			}
			if tt.wantErr != nil && !errorOccurred {
				t.Fatalf("Expected error %v did not occur", tt.wantErr)
			}
			if tt.wantErr == nil {
				printRouter(r)
			}
		})
	}
}

func buildRouter(routes []string) *Router[int] {
	r := &Router[int]{}
	for value, path := range routes {
		err := r.Add(path, 100+value)
		if err != nil {
			panic("Add path: " + path + ", error: " + err.Error())
		}
	}
	return r

}

func TestRouter_Match(t *testing.T) {
	tests := []struct {
		name    string
		routes  []string
		path    string
		want    bool
		wantRes MatchResult[int]
	}{
		{name: "nil", path: "", want: false},
		{name: "empty", routes: []string{""}, path: "", want: true, wantRes: MatchResult[int]{Value: 100}},
		{name: "empty", routes: []string{"a", "b", ""}, path: "", want: true, wantRes: MatchResult[int]{Value: 102}},
		{name: "empty-fail", routes: []string{"a", "b"}, path: "", want: false},
		{name: "static", routes: []string{"/a", "/a/b", "/a/c"}, path: "/a/b", want: true, wantRes: MatchResult[int]{Value: 101}},
		{name: "static-fail", routes: []string{"/a", "/a/b", "/a/c"}, path: "/a/d", want: false},
		{name: "param", routes: []string{"/a/:param1", "/a/:param1/:param2", "/b/:param3"}, path: "/a/d", want: true, wantRes: MatchResult[int]{Params: Params{{Key: "param1", Value: "d"}}, Value: 100}},
		{name: "param-1", routes: []string{"/a/:param1", "/a/:param1/:param2", "/b/:param3"}, path: "/a/b/c", want: true, wantRes: MatchResult[int]{Params: Params{{Key: "param1", Value: "b"}, {Key: "param2", Value: "c"}}, Value: 101}},
		{name: "param-2", routes: []string{"/a/:param1", "/a/:param1/:param2", "/b/:param3"}, path: "/b/123", want: true, wantRes: MatchResult[int]{Params: Params{{Key: "param3", Value: "123"}}, Value: 102}},
		{name: "param-prefix", routes: []string{"/a/a-:param1", "/a/b-:param2"}, path: "/a/b-123", want: true, wantRes: MatchResult[int]{Params: Params{{Key: "param2", Value: "123"}}, Value: 101}},
		{name: "param-fail", routes: []string{"/a/:param1"}, path: "/b/d", want: false},
		{name: "param-fail-1", routes: []string{":param1"}, path: "", want: false},
		{name: "param-fail-2", routes: []string{"/a/:param1"}, path: "/a/d/", want: false},
		{name: "trailing", routes: []string{"/a/:param1/*", "/b/*"}, path: "/a/b/c", want: true, wantRes: MatchResult[int]{Params: Params{{Key: "param1", Value: "b"}, {Key: "*", Value: "c"}}, Value: 100}},
		{name: "trailing-1", routes: []string{"/a/:param1/*", "/b/*"}, path: "/b/123", want: true, wantRes: MatchResult[int]{Params: Params{{Key: "*", Value: "123"}}, Value: 101}},
		{name: "trailing-2", routes: []string{"*"}, path: "", want: true, wantRes: MatchResult[int]{Params: Params{{Key: "*", Value: ""}}, Value: 100}},
		{name: "trailing-3", routes: []string{"/a/*"}, path: "/a/", want: true, wantRes: MatchResult[int]{Params: Params{{Key: "*", Value: ""}}, Value: 100}},
		{name: "trailing-4", routes: []string{"/a/", "/a/*"}, path: "/a/", want: true, wantRes: MatchResult[int]{Value: 100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := buildRouter(tt.routes)
			var res MatchResult[int]
			if got := r.Match(tt.path, &res); got != tt.want {
				t.Errorf("Router.Match() = %v, want %v", got, tt.want)
				return
			}
			if tt.want && !reflect.DeepEqual(res, tt.wantRes) {
				t.Errorf("MatchResult got %+v, want %+v", res, tt.wantRes)
			}
		})
	}
}

var githubRoutes = []*struct {
	Method string
	Path   string
}{
	// OAuth Authorizations
	{"GET", "/authorizations"},
	{"GET", "/authorizations/:id"},
	{"POST", "/authorizations"},
	{"DELETE", "/authorizations/:id"},
	{"GET", "/applications/:client_id/tokens/:access_token"},
	{"DELETE", "/applications/:client_id/tokens"},
	{"DELETE", "/applications/:client_id/tokens/:access_token"},

	// Activity
	{"GET", "/events"},
	{"GET", "/repos/:owner/:repo/events"},
	{"GET", "/networks/:owner/:repo/events"},
	{"GET", "/orgs/:org/events"},
	{"GET", "/users/:user/received_events"},
	{"GET", "/users/:user/received_events/public"},
	{"GET", "/users/:user/events"},
	{"GET", "/users/:user/events/public"},
	{"GET", "/users/:user/events/orgs/:org"},
	{"GET", "/feeds"},
	{"GET", "/notifications"},
	{"GET", "/repos/:owner/:repo/notifications"},
	{"PUT", "/notifications"},
	{"PUT", "/repos/:owner/:repo/notifications"},
	{"GET", "/notifications/threads/:id"},
	{"GET", "/notifications/threads/:id/subscription"},
	{"PUT", "/notifications/threads/:id/subscription"},
	{"DELETE", "/notifications/threads/:id/subscription"},
	{"GET", "/repos/:owner/:repo/stargazers"},
	{"GET", "/users/:user/starred"},
	{"GET", "/user/starred"},
	{"GET", "/user/starred/:owner/:repo"},
	{"PUT", "/user/starred/:owner/:repo"},
	{"DELETE", "/user/starred/:owner/:repo"},
	{"GET", "/repos/:owner/:repo/subscribers"},
	{"GET", "/users/:user/subscriptions"},
	{"GET", "/user/subscriptions"},
	{"GET", "/repos/:owner/:repo/subscription"},
	{"PUT", "/repos/:owner/:repo/subscription"},
	{"DELETE", "/repos/:owner/:repo/subscription"},
	{"GET", "/user/subscriptions/:owner/:repo"},
	{"PUT", "/user/subscriptions/:owner/:repo"},
	{"DELETE", "/user/subscriptions/:owner/:repo"},

	// Gists
	{"GET", "/users/:user/gists"},
	{"GET", "/gists"},
	{"GET", "/gists/:id"},
	{"POST", "/gists"},
	{"DELETE", "/gists/:id/star"},
	{"GET", "/gists/:id/star"},
	{"POST", "/gists/:id/forks"},
	{"DELETE", "/gists/:id"},

	// Git Data
	{"GET", "/repos/:owner/:repo/git/blobs/:sha"},
	{"POST", "/repos/:owner/:repo/git/blobs"},
	{"GET", "/repos/:owner/:repo/git/commits/:sha"},
	{"POST", "/repos/:owner/:repo/git/commits"},
	{"GET", "/repos/:owner/:repo/git/refs"},
	{"POST", "/repos/:owner/:repo/git/refs"},
	{"GET", "/repos/:owner/:repo/git/tags/:sha"},
	{"POST", "/repos/:owner/:repo/git/tags"},
	{"GET", "/repos/:owner/:repo/git/trees/:sha"},
	{"POST", "/repos/:owner/:repo/git/trees"},

	// Issues
	{"GET", "/issues"},
	{"GET", "/user/issues"},
	{"GET", "/orgs/:org/issues"},
	{"GET", "/repos/:owner/:repo/issues"},
	{"GET", "/repos/:owner/:repo/issues/:number"},
	{"POST", "/repos/:owner/:repo/issues"},
	{"GET", "/repos/:owner/:repo/assignees"},
	{"GET", "/repos/:owner/:repo/assignees/:assignee"},
	{"GET", "/repos/:owner/:repo/issues/:number/comments"},
	{"POST", "/repos/:owner/:repo/issues/:number/comments"},
	{"GET", "/repos/:owner/:repo/issues/:number/events"},
	{"GET", "/repos/:owner/:repo/labels"},
	{"GET", "/repos/:owner/:repo/labels/:name"},
	{"POST", "/repos/:owner/:repo/labels"},
	{"DELETE", "/repos/:owner/:repo/labels/:name"},
	{"GET", "/repos/:owner/:repo/issues/:number/labels"},
	{"POST", "/repos/:owner/:repo/issues/:number/labels"},
	{"DELETE", "/repos/:owner/:repo/issues/:number/labels/:name"},
	{"PUT", "/repos/:owner/:repo/issues/:number/labels"},
	{"DELETE", "/repos/:owner/:repo/issues/:number/labels"},
	{"GET", "/repos/:owner/:repo/milestones/:number/labels"},
	{"GET", "/repos/:owner/:repo/milestones"},
	{"GET", "/repos/:owner/:repo/milestones/:number"},
	{"POST", "/repos/:owner/:repo/milestones"},
	{"DELETE", "/repos/:owner/:repo/milestones/:number"},

	// Miscellaneous
	{"GET", "/emojis"},
	{"GET", "/gitignore/templates"},
	{"GET", "/gitignore/templates/:name"},
	{"POST", "/markdown"},
	{"POST", "/markdown/raw"},
	{"GET", "/meta"},
	{"GET", "/rate_limit"},

	// Organizations
	{"GET", "/users/:user/orgs"},
	{"GET", "/user/orgs"},
	{"GET", "/orgs/:org"},
	{"GET", "/orgs/:org/members"},
	{"GET", "/orgs/:org/members/:user"},
	{"DELETE", "/orgs/:org/members/:user"},
	{"GET", "/orgs/:org/public_members"},
	{"GET", "/orgs/:org/public_members/:user"},
	{"PUT", "/orgs/:org/public_members/:user"},
	{"DELETE", "/orgs/:org/public_members/:user"},
	{"GET", "/orgs/:org/teams"},
	{"GET", "/teams/:id"},
	{"POST", "/orgs/:org/teams"},
	{"DELETE", "/teams/:id"},
	{"GET", "/teams/:id/members"},
	{"GET", "/teams/:id/members/:user"},
	{"PUT", "/teams/:id/members/:user"},
	{"DELETE", "/teams/:id/members/:user"},
	{"GET", "/teams/:id/repos"},
	{"GET", "/teams/:id/repos/:owner/:repo"},
	{"PUT", "/teams/:id/repos/:owner/:repo"},
	{"DELETE", "/teams/:id/repos/:owner/:repo"},
	{"GET", "/user/teams"},

	// Pull Requests
	{"GET", "/repos/:owner/:repo/pulls"},
	{"GET", "/repos/:owner/:repo/pulls/:number"},
	{"POST", "/repos/:owner/:repo/pulls"},
	{"GET", "/repos/:owner/:repo/pulls/:number/commits"},
	{"GET", "/repos/:owner/:repo/pulls/:number/files"},
	{"GET", "/repos/:owner/:repo/pulls/:number/merge"},
	{"PUT", "/repos/:owner/:repo/pulls/:number/merge"},
	{"GET", "/repos/:owner/:repo/pulls/:number/comments"},
	{"PUT", "/repos/:owner/:repo/pulls/:number/comments"},

	// Repositories
	{"GET", "/user/repos"},
	{"GET", "/users/:user/repos"},
	{"GET", "/orgs/:org/repos"},
	{"GET", "/repositories"},
	{"POST", "/user/repos"},
	{"POST", "/orgs/:org/repos"},
	{"GET", "/repos/:owner/:repo"},
	{"GET", "/repos/:owner/:repo/contributors"},
	{"GET", "/repos/:owner/:repo/languages"},
	{"GET", "/repos/:owner/:repo/teams"},
	{"GET", "/repos/:owner/:repo/tags"},
	{"GET", "/repos/:owner/:repo/branches"},
	{"GET", "/repos/:owner/:repo/branches/:branch"},
	{"DELETE", "/repos/:owner/:repo"},
	{"GET", "/repos/:owner/:repo/collaborators"},
	{"GET", "/repos/:owner/:repo/collaborators/:user"},
	{"PUT", "/repos/:owner/:repo/collaborators/:user"},
	{"DELETE", "/repos/:owner/:repo/collaborators/:user"},
	{"GET", "/repos/:owner/:repo/comments"},
	{"GET", "/repos/:owner/:repo/commits/:sha/comments"},
	{"POST", "/repos/:owner/:repo/commits/:sha/comments"},
	{"GET", "/repos/:owner/:repo/comments/:id"},
	{"DELETE", "/repos/:owner/:repo/comments/:id"},
	{"GET", "/repos/:owner/:repo/commits"},
	{"GET", "/repos/:owner/:repo/commits/:sha"},
	{"GET", "/repos/:owner/:repo/readme"},
	{"GET", "/repos/:owner/:repo/keys"},
	{"GET", "/repos/:owner/:repo/keys/:id"},
	{"POST", "/repos/:owner/:repo/keys"},
	{"DELETE", "/repos/:owner/:repo/keys/:id"},
	{"GET", "/repos/:owner/:repo/downloads"},
	{"GET", "/repos/:owner/:repo/downloads/:id"},
	{"DELETE", "/repos/:owner/:repo/downloads/:id"},
	{"GET", "/repos/:owner/:repo/forks"},
	{"POST", "/repos/:owner/:repo/forks"},
	{"GET", "/repos/:owner/:repo/hooks"},
	{"GET", "/repos/:owner/:repo/hooks/:id"},
	{"POST", "/repos/:owner/:repo/hooks"},
	{"POST", "/repos/:owner/:repo/hooks/:id/tests"},
	{"DELETE", "/repos/:owner/:repo/hooks/:id"},
	{"POST", "/repos/:owner/:repo/merges"},
	{"GET", "/repos/:owner/:repo/releases"},
	{"GET", "/repos/:owner/:repo/releases/:id"},
	{"POST", "/repos/:owner/:repo/releases"},
	{"DELETE", "/repos/:owner/:repo/releases/:id"},
	{"GET", "/repos/:owner/:repo/releases/:id/assets"},
	{"GET", "/repos/:owner/:repo/stats/contributors"},
	{"GET", "/repos/:owner/:repo/stats/commit_activity"},
	{"GET", "/repos/:owner/:repo/stats/code_frequency"},
	{"GET", "/repos/:owner/:repo/stats/participation"},
	{"GET", "/repos/:owner/:repo/stats/punch_card"},
	{"GET", "/repos/:owner/:repo/statuses/:ref"},
	{"POST", "/repos/:owner/:repo/statuses/:ref"},

	// Search
	{"GET", "/search/repositories"},
	{"GET", "/search/code"},
	{"GET", "/search/issues"},
	{"GET", "/search/users"},
	{"GET", "/legacy/issues/search/:owner/:repository/:state/:keyword"},
	{"GET", "/legacy/repos/search/:keyword"},
	{"GET", "/legacy/user/search/:keyword"},
	{"GET", "/legacy/user/email/:email"},

	// Users
	{"GET", "/users/:user"},
	{"GET", "/user"},
	{"GET", "/users"},
	{"GET", "/user/emails"},
	{"POST", "/user/emails"},
	{"DELETE", "/user/emails"},
	{"GET", "/users/:user/followers"},
	{"GET", "/user/followers"},
	{"GET", "/users/:user/following"},
	{"GET", "/user/following"},
	{"GET", "/user/following/:user"},
	{"GET", "/users/:user/following/:target_user"},
	{"PUT", "/user/following/:user"},
	{"DELETE", "/user/following/:user"},
	{"GET", "/users/:user/keys"},
	{"GET", "/user/keys"},
	{"GET", "/user/keys/:id"},
	{"POST", "/user/keys"},
	{"DELETE", "/user/keys/:id"},
}

func BenchmarkGithubRoutes(b *testing.B) {
	r := &Router[int]{}
	for i, route := range githubRoutes {
		err := r.Add(route.Path, 100+i)
		if err != nil {
			panic(err)
		}
	}
	res := MatchResult[int]{Params: make(Params, 0, 10)}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, route := range githubRoutes {
			res.Params = res.Params[:0]
			ok := r.Match(route.Path, &res)
			if !ok {
				panic("bad")
			}
		}
	}
}
