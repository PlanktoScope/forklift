package git

import (
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"
)

func Remotes(local string) (remotes []*git.Remote, err error) {
	repo, err := git.PlainOpen(local)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't open %s as git repo", local)
	}
	return repo.Remotes()
}

func Head(local string) (ref *plumbing.Reference, err error) {
	repo, err := git.PlainOpen(local)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't open %s as git repo", local)
	}
	return repo.Head()
}

func Refs(local string) (refs []*plumbing.Reference, err error) {
	repo, err := git.PlainOpen(local)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't open %s as git repo", local)
	}
	refsIter, err := repo.References()
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't list refs of %s", local)
	}
	defer refsIter.Close()

	refs = make([]*plumbing.Reference, 0)
	err = refsIter.ForEach(func(ref *plumbing.Reference) error {
		refs = append(refs, ref)
		return nil
	})
	return refs, err
}

func FilterBranches(refs []*plumbing.Reference) []*plumbing.Reference {
	branches := make([]*plumbing.Reference, 0, len(refs))
	for _, ref := range refs {
		if !ref.Name().IsBranch() {
			continue
		}
		branches = append(branches, ref)
	}
	return branches
}

func FilterTags(refs []*plumbing.Reference) []*plumbing.Reference {
	tags := make([]*plumbing.Reference, 0, len(refs))
	for _, ref := range refs {
		if !ref.Name().IsTag() {
			continue
		}
		tags = append(tags, ref)
	}
	return tags
}

func FilterRemotes(refs []*plumbing.Reference) []*plumbing.Reference {
	remotes := make([]*plumbing.Reference, 0, len(refs))
	for _, ref := range refs {
		if !ref.Name().IsRemote() {
			continue
		}
		remotes = append(remotes, ref)
	}
	return remotes
}

func AnnotateRefName(name plumbing.ReferenceName) string {
	b := strings.Builder{}
	switch {
	case name.IsBranch():
		b.WriteString("(branch) ")
	case name.IsTag():
		b.WriteString("(tag) ")
	case name.IsRemote():
		b.WriteString("(remote) ")
	}
	b.WriteString(name.Short())
	return b.String()
}

func StringifyRef(ref *plumbing.Reference) string {
	b := strings.Builder{}
	b.WriteString(AnnotateRefName(ref.Name()))
	b.WriteString(" -> ")
	switch ref.Type() {
	default:
		b.WriteString("(invalid)")
	case plumbing.HashReference:
		b.WriteString("(commit) ")
		b.WriteString(AbbreviateHash(ref.Hash()))
	case plumbing.SymbolicReference:
		b.WriteString(AnnotateRefName(ref.Target()))
	}
	return b.String()
}
