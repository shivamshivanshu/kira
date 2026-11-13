package core

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func (s *Store) DoctorReport(cfg *datamodel.Config) (*doctor.Report, error) {
	files, err := s.ticketFiles()
	if err != nil {
		return nil, err
	}
	strays, err := s.fs().StrayFilenames()
	if err != nil {
		return nil, err
	}
	return doctor.Run(cfg, files, strays, s.doctorEnv()), nil
}

func (s *Store) ValidateFiles(cfg *datamodel.Config, args []string) (*doctor.Report, error) {
	storeFiles, err := s.ticketFiles()
	if err != nil {
		return nil, err
	}
	targets := make([]doctor.File, 0, len(args))
	for _, arg := range args {
		if s.isProjectConfig(arg) {
			return nil, errx.User("%s is the project config, not a ticket file; use kira doctor to check it", arg)
		}
		if fi, err := os.Stat(arg); err == nil && fi.Mode().IsRegular() {
			data, err := os.ReadFile(arg)
			if err != nil {
				return nil, errx.User("reading %s: %v", arg, err)
			}
			targets = append(targets, doctor.File{Path: arg, Content: string(data)})
			continue
		}
		path, content, err := s.ResolveItemFile(cfg, arg)
		if err != nil {
			return nil, errx.User("%q is neither a file nor a resolvable ticket id", arg)
		}
		targets = append(targets, doctor.File{Path: path, Content: content})
	}
	return doctor.Validate(cfg, storeFiles, targets), nil
}

func (s *Store) isProjectConfig(arg string) bool {
	cfgPath, cfgErr := filepath.Abs(filepath.Join(s.root, storage.ConfigRelPath))
	argPath, argErr := filepath.Abs(arg)
	return cfgErr == nil && argErr == nil && cfgPath == argPath
}

func (s *Store) ticketFiles() ([]doctor.File, error) {
	raw, err := s.fs().RawItems()
	if err != nil {
		return nil, err
	}
	files := make([]doctor.File, len(raw))
	for i, r := range raw {
		files[i] = doctor.File{Path: r.Name, Content: r.Content}
	}
	return files, nil
}

func (s *Store) doctorEnv() doctor.Env {
	env := doctor.Env{GitInstalled: gitx.Installed(), LegacySchemaDirPresent: s.legacySchemaDirPresent()}
	if !env.GitInstalled {
		return env
	}
	env.MissingOptionalBins = missingBinaries("rg")
	repo := s.repo()
	env.InsideWorkTree = repo.InsideWorkTree() == nil
	if !env.InsideWorkTree {
		return env
	}
	env.Freshness = doctor.ResolveFreshness(indexFreshness{store: s.fs(), repo: repo})
	return env
}

func (s *Store) legacySchemaDirPresent() bool {
	fi, err := os.Stat(s.fs().LegacySchemaDir())
	return err == nil && fi.IsDir()
}

type indexFreshness struct {
	store *storage.FS
	repo  gitx.Repo
}

func (r indexFreshness) Freshness() (doctor.Freshness, error) {
	rep, err := index.Probe(r.store, r.repo)
	if err != nil {
		return doctor.Freshness{}, err
	}
	return doctor.Freshness{Built: rep.Built, Fresh: rep.Fresh, Reason: rep.Reason}, nil
}

func missingBinaries(names ...string) []string {
	var out []string
	for _, n := range names {
		if _, err := exec.LookPath(n); err != nil {
			out = append(out, n)
		}
	}
	return out
}
