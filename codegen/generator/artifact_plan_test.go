package generator

import (
	"errors"
	"io"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"weak"

	"github.com/CaliLuke/loom/codegen"
	"github.com/stretchr/testify/require"
)

type scheduledSection struct {
	name string
	run  func(io.Writer) error
}

func TestArtifactPlanAdmission(t *testing.T) {
	for _, tc := range []struct {
		name     string
		paths    []string
		removals []string
		command  string
	}{
		{name: "escaping write", paths: []string{"good.txt", "../outside.txt"}},
		{name: "root write", paths: []string{"good.txt", "."}},
		{name: "absolute write", paths: []string{filepath.Join(t.TempDir(), "absolute.txt")}},
		{name: "case collision", paths: []string{"A.txt", "a.txt"}},
		{name: "unicode case collision", paths: []string{"Σ.txt", "ς.txt"}},
		{name: "file parent", paths: []string{"a", "a/b.txt"}},
		{name: "case parent", paths: []string{"A", "a/b.txt"}},
		{name: "intervening sibling", paths: []string{"a", "a-b", "a/b.txt"}},
		{name: "reserved manifest", paths: []string{"gen/loom.json"}, command: "gen"},
		{name: "case manifest", paths: []string{"GEN/LOOM.JSON"}, command: "gen"},
		{name: "manifest child", paths: []string{"gen/loom.json/child"}, command: "gen"},
		{name: "manifest parent", paths: []string{"gen"}, command: "gen"},
		{name: "escaping removal", paths: []string{"good.txt"}, removals: []string{"../old.txt"}},
		{name: "root removal", paths: []string{"good.txt"}, removals: []string{"."}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var called atomic.Int32
			files := make([]*codegen.File, 0, len(tc.paths))
			for _, path := range tc.paths {
				files = append(files, &codegen.File{
					Path: path, RemovePaths: tc.removals,
					Sections: []codegen.Section{scheduledSection{name: "body", run: func(io.Writer) error {
						called.Add(1)
						return nil
					}}},
					FinalizeFunc: func(string) error {
						called.Add(1)
						return nil
					},
				})
			}
			plan, err := planArtifacts(t.TempDir(), tc.command, "example/design", "digest", files, false)
			require.Error(t, err)
			require.Nil(t, plan)
			require.Zero(t, called.Load())
		})
	}
}

func TestArtifactPlanEmptyAndIndependentPaths(t *testing.T) {
	dir := t.TempDir()
	empty, err := planArtifacts(dir, "example", "example/design", "", []*codegen.File{nil}, false)
	require.NoError(t, err)
	written, err := empty.render(dir, 2, false)
	require.NoError(t, err)
	require.Empty(t, written)
	files := []*codegen.File{
		{Path: "a.txt", Sections: []codegen.Section{codegen.NewRawSection("one", "a")}},
		{Path: "nested/../a.txt", Sections: []codegen.Section{codegen.NewRawSection("two", "b")}},
		{Path: "abc.txt", Sections: []codegen.Section{codegen.NewRawSection("one", "c")}},
	}
	plan, err := planArtifacts(dir, "example", "example/design", "", files, false)
	require.NoError(t, err)
	require.Len(t, plan.files, 2)
	written, err = plan.render(dir, 2, false)
	require.NoError(t, err)
	require.Len(t, written, 2)
	require.Nil(t, plan.files)
	require.NotEmpty(t, files[0].Sections, "the scheduler must not clear caller-owned sections")
}

func TestArtifactRenderStopsAndJoinsWithOrderedFailure(t *testing.T) {
	dir := t.TempDir()
	started := make(chan int, 3)
	release := []chan struct{}{make(chan struct{}), make(chan struct{})}
	first, second := errors.New("first"), errors.New("second")
	var finished atomic.Int32
	files := make([]*codegen.File, 3)
	for index := range files {
		files[index] = &codegen.File{Path: filepath.Join("out", string(rune('a'+index))+".txt"), Sections: []codegen.Section{scheduledSection{name: "body", run: func(io.Writer) error {
			started <- index
			if index >= 2 {
				return errors.New("queued work should not start")
			}
			<-release[index]
			finished.Add(1)
			if index == 0 {
				return first
			}
			return second
		}}}}
	}
	plan, err := planArtifacts(dir, "example", "example/design", "", files, false)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		_, err := plan.render(dir, 2, false)
		done <- err
	}()
	require.ElementsMatch(t, []int{0, 1}, []int{<-started, <-started})
	close(release[1])
	close(release[0])
	require.ErrorIs(t, <-done, first)
	require.EqualValues(t, 2, finished.Load())
	require.Empty(t, started)
	require.Nil(t, plan.files)
}

func TestGenerateRenderOptions(t *testing.T) {
	config, err := generationConfiguration(nil)
	require.NoError(t, err)
	require.Equal(t, min(4, runtime.GOMAXPROCS(0)), config.workers)
	config, err = generationConfiguration([]GenerateOption{WithRenderWorkers(1), WithRenderWorkers(3)})
	require.NoError(t, err)
	require.Equal(t, 3, config.workers)
	for _, option := range []GenerateOption{nil, WithRenderWorkers(0), WithRenderWorkers(-1)} {
		_, err := Generate(t.TempDir(), "gen", "invalid design", false, option)
		require.ErrorContains(t, err, "render workers")
	}
}

func TestArtifactRenderBoundsSuccessfulWork(t *testing.T) {
	dir := t.TempDir()
	started := make(chan int, 5)
	release := make([]chan struct{}, 5)
	var active, maximum atomic.Int32
	files := make([]*codegen.File, 5)
	for index := range files {
		release[index] = make(chan struct{})
		files[index] = &codegen.File{
			Path: string(rune('a'+index)) + ".txt",
			Sections: []codegen.Section{scheduledSection{name: "body", run: func(io.Writer) error {
				current := active.Add(1)
				for old := maximum.Load(); current > old; old = maximum.Load() {
					if maximum.CompareAndSwap(old, current) {
						break
					}
				}
				started <- index
				<-release[index]
				active.Add(-1)
				return nil
			}}},
		}
	}
	plan, err := planArtifacts(dir, "example", "example/design", "", files, false)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		_, err := plan.render(dir, 2, false)
		done <- err
	}()
	first, second := <-started, <-started
	require.ElementsMatch(t, []int{0, 1}, []int{first, second})
	close(release[first])
	for index := 2; index < len(files); index++ {
		next := <-started
		require.Equal(t, index, next)
		close(release[next])
	}
	close(release[second])
	require.NoError(t, <-done)
	require.EqualValues(t, 2, maximum.Load())
	require.Zero(t, active.Load())
	require.Nil(t, plan.files)
}

func TestArtifactPlanReleasesCompletedFiles(t *testing.T) {
	for _, fail := range []bool{false, true} {
		plan, file := completedArtifactPlan(t, fail)
		for range 10 {
			runtime.GC()
		}
		require.Nil(t, file.Value(), "a retained completed plan must not retain file render data")
		runtime.KeepAlive(plan)
	}
}

func (s scheduledSection) SectionName() string {
	return s.name
}

func (s scheduledSection) Write(w io.Writer) error {
	return s.run(w)
}

//go:noinline
func completedArtifactPlan(t *testing.T, fail bool) (*artifactPlan, weak.Pointer[codegen.File]) {
	t.Helper()
	dir := t.TempDir()
	file := &codegen.File{Path: "file.txt", Sections: []codegen.Section{scheduledSection{name: "body", run: func(io.Writer) error {
		if fail {
			return errors.New("render failure")
		}
		return nil
	}}}}
	pointer := weak.Make(file)
	plan, err := planArtifacts(dir, "example", "example/design", "", []*codegen.File{file}, false)
	require.NoError(t, err)
	_, err = plan.render(dir, 1, false)
	if fail {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}
	return plan, pointer
}
