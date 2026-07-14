package core

import (
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const (
	subjectNumbersCap = 5
	commitTrailerCap  = 20
)

func (s *Store) CommitKira(cfg *datamodel.Config) (*datamodel.CommitResult, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	release, err := s.fs().Lock()
	if err != nil {
		return nil, err
	}
	defer release()

	repo := s.repo()
	dirty, err := repo.DirtyPaths(storage.DirName)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	if len(dirty) == 0 {
		return nil, errx.User("no kira changes to commit").
			WithHint("nothing is dirty under %s/; mutate a ticket first", storage.DirName)
	}

	numbers, body := s.summarizeDirty(cfg, dirty)
	count := len(numbers)
	if count == 0 {
		count = len(dirty)
	}
	subject := renderCommitSubject(commitSubjectTemplate(cfg, len(numbers)), count, numbers)

	if err := repo.Stage(storage.DirName); err != nil {
		return nil, errx.User("%s", err)
	}
	parts := []string{subject, strings.Join(body, "\n")}
	if trailers := commitTrailers(cfg.Commit.Trailer, numbers); trailers != "" {
		parts = append(parts, trailers)
	}
	if err := repo.CommitScoped([]string{storage.DirName}, parts...); err != nil {
		return nil, errx.User("%s", err)
	}
	sha, _ := repo.Output("rev-parse", "HEAD")
	if numbers == nil {
		numbers = []string{}
	}
	return &datamodel.CommitResult{
		Committed: true,
		SHA:       sha,
		Subject:   subject,
		Files:     len(dirty),
		Items:     numbers,
	}, nil
}

func (s *Store) summarizeDirty(cfg *datamodel.Config, dirty []string) (numbers, body []string) {
	byID := map[string]*datamodel.Item{}
	if ld, err := s.load(cfg); err == nil {
		byID = byULID(ld.items)
	}
	for _, p := range dirty {
		if !storage.IsItemPath(p) {
			body = append(body, p)
			continue
		}
		ulid := strings.TrimSuffix(path.Base(p), path.Ext(p))
		it, ok := byID[ulid]
		if !ok {
			body = append(body, p)
			continue
		}
		numbers = append(numbers, it.Number)
		body = append(body, it.Number+" "+it.Title)
	}
	return numbers, body
}

func commitSubjectTemplate(cfg *datamodel.Config, items int) string {
	if cfg.UserCommitSubject != "" {
		return cfg.UserCommitSubject
	}
	if items == 0 {
		return cfg.Commit.SubjectPrefix + "update {count} files"
	}
	return cfg.Commit.SubjectPrefix + "update {count} items"
}

func renderCommitSubject(template string, count int, numbers []string) string {
	return strings.NewReplacer(
		"{count}", strconv.Itoa(count),
		"{numbers}", cappedNumbers(numbers),
		"{date}", time.Now().Format("2006-01-02"),
	).Replace(template)
}

func cappedNumbers(numbers []string) string {
	if len(numbers) <= subjectNumbersCap {
		return strings.Join(numbers, " ")
	}
	rest := strconv.Itoa(len(numbers) - subjectNumbersCap)
	return strings.Join(numbers[:subjectNumbersCap], " ") + " +" + rest + " more"
}

func commitTrailers(key string, numbers []string) string {
	if len(numbers) > commitTrailerCap {
		numbers = numbers[:commitTrailerCap]
	}
	lines := make([]string, len(numbers))
	for i, n := range numbers {
		lines[i] = key + ": " + n
	}
	return strings.Join(lines, "\n")
}
