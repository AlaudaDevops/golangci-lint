package processors

import (
	"fmt"
	"path/filepath"

	"github.com/golangci/golangci-lint/v2/pkg/fsutils"
	"github.com/golangci/golangci-lint/v2/pkg/logutils"
	"github.com/golangci/golangci-lint/v2/pkg/result"
)

var _ Processor = (*PathRelativity)(nil)

// PathRelativity computes [result.Issue.RelativePath] and [result.Issue.WorkingDirectoryRelativePath],
// based on the base path.
type PathRelativity struct {
	log              logutils.Log
	basePath         string
	workingDirectory string
}

func NewPathRelativity(log logutils.Log, basePath string) (*PathRelativity, error) {
	wd, err := fsutils.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting working directory: %w", err)
	}

	// Resolve symlinks in basePath so it is consistent with the working directory
	// (which is already symlink-resolved by fsutils.Getwd) and with issue file paths
	// after symlink resolution. Without this, a symlink in the cwd (e.g. /Users/foo/code
	// -> /Volumes/disk/code on macOS) causes filepath.Rel to produce incorrect paths.
	if resolved, err := fsutils.EvalSymlinks(basePath); err == nil {
		basePath = resolved
	}

	return &PathRelativity{
		log:              log.Child(logutils.DebugKeyPathRelativity),
		basePath:         basePath,
		workingDirectory: wd,
	}, nil
}

func (*PathRelativity) Name() string {
	return "path_relativity"
}

func (p *PathRelativity) Process(issues []*result.Issue) ([]*result.Issue, error) {
	return transformIssues(issues, func(issue *result.Issue) *result.Issue {
		newIssue := *issue

		// Resolve symlinks so that file paths are consistent with basePath/workingDirectory,
		// which are already symlink-resolved via fsutils.Getwd(). Without this step, a
		// symlink in the working directory (e.g. /Users/foo/code -> /Volumes/disk/code on
		// macOS) causes filepath.Rel to produce a path with many "../" components because
		// the two roots share no common prefix.
		filePath := issue.FilePath()
		if resolved, err := fsutils.EvalSymlinks(filePath); err == nil {
			filePath = resolved
		}

		var err error
		newIssue.RelativePath, err = filepath.Rel(p.basePath, filePath)
		if err != nil {
			p.log.Warnf("Getting relative path (basepath): %v", err)
			return nil
		}

		newIssue.WorkingDirectoryRelativePath, err = filepath.Rel(p.workingDirectory, filePath)
		if err != nil {
			p.log.Warnf("Getting relative path (wd): %v", err)
			return nil
		}

		return &newIssue
	}), nil
}

func (*PathRelativity) Finish() {}
