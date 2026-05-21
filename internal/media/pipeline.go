package media

import (
	"context"
	"fmt"
	"strings"
)

type Pipeline struct {
	resolvers []MediaResolver
}

type PipelineResult struct {
	Candidates  []MediaCandidate
	Diagnostics []Diagnostic
}

type Diagnostic struct {
	ResolverName string
	Message      string
	Err          error
}

func NewPipeline(resolvers ...MediaResolver) *Pipeline {
	copied := append([]MediaResolver{}, resolvers...)
	return &Pipeline{resolvers: copied}
}

func (p *Pipeline) Resolve(ctx context.Context, post PostContext) (PipelineResult, error) {
	var result PipelineResult
	seen := map[string]struct{}{}
	for _, resolver := range p.resolvers {
		if resolver == nil {
			continue
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
		var (
			candidates []MediaCandidate
			err        error
		)
		if aware, ok := resolver.(CurrentAwareResolver); ok {
			candidates, err = aware.ResolveWithCurrent(ctx, post, result.Candidates)
		} else {
			candidates, err = resolver.Resolve(ctx, post)
		}
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				ResolverName: resolver.Name(),
				Message:      "resolver failed",
				Err:          err,
			})
			continue
		}
		for _, candidate := range candidates {
			if candidate.ResolverName == "" {
				candidate.ResolverName = resolver.Name()
			}
			if candidate.CandidateID == "" {
				candidate.CandidateID = candidateID(candidate)
			}
			key := candidateDedupKey(candidate)
			if _, ok := seen[key]; ok {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					ResolverName: resolver.Name(),
					Message:      "duplicate candidate skipped",
				})
				continue
			}
			seen[key] = struct{}{}
			result.Candidates = append(result.Candidates, candidate)
		}
	}
	return result, nil
}

func (p *Pipeline) ResolverNames() string {
	names := make([]string, 0, len(p.resolvers))
	for _, resolver := range p.resolvers {
		if resolver != nil {
			names = append(names, resolver.Name())
		}
	}
	return strings.Join(names, ", ")
}

func candidateDedupKey(candidate MediaCandidate) string {
	if candidate.URL != "" {
		return strings.Join([]string{candidate.URL, string(candidate.MediaKind)}, "\x1f")
	}
	return strings.Join([]string{
		candidate.URL,
		string(candidate.MediaKind),
		candidate.ContentRole,
		candidate.SourceField,
		fmt.Sprintf("%d", candidate.Index),
	}, "\x1f")
}

func candidateID(candidate MediaCandidate) string {
	base := candidateDedupKey(candidate)
	if base == "\x1f\x1f\x1f\x1f0" {
		base = candidate.PostID + "\x1f" + candidate.SourceField + "\x1f" + candidate.ContentRole
	}
	return "candidate_" + shortHash(base)
}

func shortHash(value string) string {
	var hash uint64 = 1469598103934665603
	for _, b := range []byte(value) {
		hash ^= uint64(b)
		hash *= 1099511628211
	}
	return fmt.Sprintf("%016x", hash)
}
