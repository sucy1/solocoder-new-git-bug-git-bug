package resolvers

import (
	"context"
	"image/color"

	"github.com/git-bug/git-bug/api/graphql/graph"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/common"
)

var _ graph.LabelResolver = &labelResolver{}

type labelResolver struct {
	cache *cache.MultiRepoCache
}

func (r labelResolver) Name(ctx context.Context, obj *common.Label) (string, error) {
	return obj.String(), nil
}

func (r labelResolver) Color(ctx context.Context, obj *common.Label) (*color.RGBA, error) {
	if r.cache != nil {
		repo, err := r.cache.DefaultRepo()
		if err == nil {
			labelColor := common.GetLabelColor(repo.AnyConfig(), *obj)
			rgba := labelColor.RGBA()
			return &rgba, nil
		}
	}

	rgba := obj.DefaultColor().RGBA()
	return &rgba, nil
}
