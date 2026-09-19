package cache

import (
	"context"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/config"
	"ecommerce-be/product/model"
)

// TTL bases for category/attribute reference data (low-write, high-read).
const (
	// CategoryDetailTTL bases single category/attribute entries.
	CategoryDetailTTL = 60 * time.Minute
	// CategoryListTTL bases versioned category/attribute lists.
	CategoryListTTL = 30 * time.Minute
	// CategoryNegativeTTL bases not-found tombstones.
	CategoryNegativeTTL = 60 * time.Second
)

// CategoryCache is the category/attribute reference-data strategy. Entries
// stay seller-scoped even for global categories: different sellers see
// different sets (global + own), so sharing keys would leak visibility.
type CategoryCache struct {
	cache    cachekit.Cache
	durable  cachekit.Durable
	flight   *cachekit.Flight
	recorder cachekit.Recorder
}

// NewCategoryCache builds the strategy. A nil cache disables all caching;
// a nil durable degrades versioned lists to unversioned TTL reads.
func NewCategoryCache(c cachekit.Cache, d cachekit.Durable, r cachekit.Recorder) *CategoryCache {
	if r == nil {
		r = cachekit.LogRecorder()
	}
	return &CategoryCache{cache: c, durable: d, flight: &cachekit.Flight{}, recorder: r}
}

// enabled reports whether category caching applies (flag-gated like product).
func (s *CategoryCache) enabled() bool {
	if s == nil || s.cache == nil {
		return false
	}
	cfg := config.Get()
	return cfg != nil && cfg.Cache.Enabled && cfg.Cache.Category
}

// fetchOne serves one JSON DTO through cache-aside. Load errors propagate
// (the fill already ran inside FetchBytes — never retried here); a nil load
// result records a tombstone.
func (s *CategoryCache) fetchOne(
	ctx context.Context,
	key, op string,
	load func(ctx context.Context) (any, error),
	decode func([]byte) (any, error),
) (any, error) {
	body, err := cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"product", op, key, CategoryDetailTTL, CategoryNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			v, err := load(ctx)
			if err != nil {
				return nil, err
			}
			if v == nil {
				return nil, nil
			}
			return cachekit.Marshal(v)
		})
	if err != nil {
		return nil, err
	}
	if out, derr := decode(body); derr == nil {
		return out, nil
	}
	// Corrupt bytes (schema drift): fall back to live instead of failing.
	return load(ctx)
}

// fetchList serves a versioned list: the durable version counter selects the
// key, so writes retire lists with one INCR and no enumeration. Without a
// durable handle it degrades to an unversioned TTL read (still correct,
// bounded by the list TTL).
func (s *CategoryCache) fetchList(
	ctx context.Context,
	sellerScope, family, qualifier, op string,
	load func(ctx context.Context) (any, error),
	decode func([]byte) (any, error),
) (any, error) {
	key := sellerScope + ":" + family + qualifier
	if s.durable != nil {
		if ver, err := s.currentVersion(ctx, sellerScope, family); err == nil {
			key = sellerScope + ":" + family + ":v" + ver + qualifier
		}
	}
	body, err := cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"product", op, key, CategoryListTTL, CategoryNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			v, err := load(ctx)
			if err != nil {
				return nil, err
			}
			if v == nil {
				return nil, nil
			}
			return cachekit.Marshal(v)
		})
	if err != nil {
		return nil, err
	}
	if out, derr := decode(body); derr == nil {
		return out, nil
	}
	return load(ctx)
}

// currentVersion reads the durable list version ("0" when never bumped —
// all readers agree, which is all that matters).
func (s *CategoryCache) currentVersion(ctx context.Context, sellerScope, family string) (string, error) {
	raw, err := s.durable.Get(ctx, sellerScope+":"+family+":ver")
	if err != nil || len(raw) == 0 {
		return "0", err
	}
	if _, err := cachekit.ParseVersion(raw); err != nil {
		return "0", err
	}
	return string(raw), nil
}

// bumpVersion advances the durable list-version counter (best-effort; the
// list TTL bounds any miss, per pre-spec §5.4).
func (s *CategoryCache) bumpVersion(ctx context.Context, sellerID uint, family string) {
	if s.durable == nil {
		return
	}
	verKey, err := cachekit.VersionKey(sellerID, family)
	if err != nil {
		return
	}
	_, _ = cachekit.BumpVersion(ctx, s.durable, verKey)
}

// decodeCategories unmarshals a categories list payload.
func decodeCategories(b []byte) (any, error) {
	var out model.CategoriesResponse
	return &out, cachekit.Unmarshal(b, &out)
}

// GetCategory serves one category through cache-aside (seller-scoped).
func (s *CategoryCache) GetCategory(
	ctx context.Context,
	sellerID *uint,
	id uint,
	load func(ctx context.Context) (*model.CategoryResponse, error),
) (*model.CategoryResponse, error) {
	if !s.enabled() || sellerID == nil {
		return load(ctx)
	}
	key, err := cachekit.BuildSellerKey(*sellerID, "category", uintToString(id))
	if err != nil {
		return load(ctx)
	}
	v, err := s.fetchOne(ctx, key, "category-detail", func(ctx context.Context) (any, error) {
		return load(ctx)
	}, func(b []byte) (any, error) {
		var out model.CategoryResponse
		return &out, cachekit.Unmarshal(b, &out)
	})
	if err != nil {
		return nil, err
	}
	out, _ := v.(*model.CategoryResponse)
	return out, nil
}

// GetAllCategories serves the full seller-visible category set (versioned).
func (s *CategoryCache) GetAllCategories(
	ctx context.Context,
	sellerID *uint,
	load func(ctx context.Context) (*model.CategoriesResponse, error),
) (*model.CategoriesResponse, error) {
	if !s.enabled() || sellerID == nil {
		return load(ctx)
	}
	scope := "seller:" + uintToString(*sellerID)
	v, err := s.fetchList(ctx, scope, "categories", ":all", "category-list",
		func(ctx context.Context) (any, error) {
			return load(ctx)
		}, decodeCategories)
	if err != nil {
		return nil, err
	}
	out, _ := v.(*model.CategoriesResponse)
	return out, nil
}

// GetCategoriesByParent serves one parent's children (versioned, small list).
func (s *CategoryCache) GetCategoriesByParent(
	ctx context.Context,
	sellerID *uint,
	parentID uint,
	load func(ctx context.Context) (*model.CategoriesResponse, error),
) (*model.CategoriesResponse, error) {
	if !s.enabled() || sellerID == nil {
		return load(ctx)
	}
	scope := "seller:" + uintToString(*sellerID)
	v, err := s.fetchList(ctx, scope, "categories", ":parent:"+uintToString(parentID),
		"category-parent", func(ctx context.Context) (any, error) {
			return load(ctx)
		}, decodeCategories)
	if err != nil {
		return nil, err
	}
	out, _ := v.(*model.CategoriesResponse)
	return out, nil
}

// attributeListVersion reads the durable global attribute-list version.
// Attribute definitions are platform-wide, so one INCR retires every
// seller's `attributes:all:v{ver}` key (pre-spec §4.5 / §5.7).
func (s *CategoryCache) attributeListVersion(ctx context.Context) string {
	if s.durable == nil {
		return "0"
	}
	raw, err := s.durable.Get(ctx, "attributes:all:ver")
	if err != nil || len(raw) == 0 {
		return "0"
	}
	if _, err := cachekit.ParseVersion(raw); err != nil {
		return "0"
	}
	return string(raw)
}

// GetAttribute serves one attribute definition through cache-aside.
func (s *CategoryCache) GetAttribute(
	ctx context.Context,
	sellerID *uint,
	id uint,
	load func(ctx context.Context) (*model.AttributeDefinitionResponse, error),
) (*model.AttributeDefinitionResponse, error) {
	if !s.enabled() || sellerID == nil {
		return load(ctx)
	}
	key, err := cachekit.BuildSellerKey(*sellerID, "attribute", uintToString(id))
	if err != nil {
		return load(ctx)
	}
	v, err := s.fetchOne(ctx, key, "attribute-detail", func(ctx context.Context) (any, error) {
		return load(ctx)
	}, func(b []byte) (any, error) {
		var out model.AttributeDefinitionResponse
		return &out, cachekit.Unmarshal(b, &out)
	})
	if err != nil {
		return nil, err
	}
	out, _ := v.(*model.AttributeDefinitionResponse)
	return out, nil
}

// GetAllAttributes serves the full attribute-definition list (versioned with
// the global attributes:all counter so admin writes retire every seller).
func (s *CategoryCache) GetAllAttributes(
	ctx context.Context,
	sellerID *uint,
	load func(ctx context.Context) (*model.AttributeDefinitionsResponse, error),
) (*model.AttributeDefinitionsResponse, error) {
	if !s.enabled() || sellerID == nil {
		return load(ctx)
	}
	ver := s.attributeListVersion(ctx)
	key, err := cachekit.BuildSellerKey(*sellerID, "attributes", "all", "v"+ver)
	if err != nil {
		return load(ctx)
	}
	body, err := cachekit.FetchBytes(ctx, s.cache, s.recorder, s.flight,
		"product", "attribute-list", key, CategoryListTTL, CategoryNegativeTTL,
		func(ctx context.Context) ([]byte, error) {
			v, err := load(ctx)
			if err != nil {
				return nil, err
			}
			if v == nil {
				return nil, nil
			}
			return cachekit.Marshal(v)
		})
	if err != nil {
		return nil, err
	}
	var out model.AttributeDefinitionsResponse
	if err := cachekit.Unmarshal(body, &out); err != nil {
		return load(ctx)
	}
	return &out, nil
}

// GetCategoryAttrs serves inherited attribute definitions for one category.
// Versioned with the categories family: link/unlink retires them.
func (s *CategoryCache) GetCategoryAttrs(
	ctx context.Context,
	sellerID *uint,
	categoryID uint,
	load func(ctx context.Context) (model.AttributeDefinitionsResponse, error),
) (model.AttributeDefinitionsResponse, error) {
	var zero model.AttributeDefinitionsResponse
	if !s.enabled() || sellerID == nil {
		return load(ctx)
	}
	scope := "seller:" + uintToString(*sellerID)
	v, err := s.fetchList(ctx, scope, "categories", ":attrs:"+uintToString(categoryID),
		"category-attrs", func(ctx context.Context) (any, error) {
			return load(ctx)
		}, func(b []byte) (any, error) {
			var out model.AttributeDefinitionsResponse
			return out, cachekit.Unmarshal(b, &out)
		})
	if err != nil {
		return zero, err
	}
	out, _ := v.(model.AttributeDefinitionsResponse)
	return out, nil
}

// InvalidateCategory removes one category entry and retires all category
// lists. Call AFTER DB commit.
func (s *CategoryCache) InvalidateCategory(ctx context.Context, sellerID, categoryID uint) {
	if s == nil || s.cache == nil {
		return
	}
	if k, err := cachekit.BuildSellerKey(sellerID, "category", uintToString(categoryID)); err == nil {
		_ = s.cache.Del(ctx, k)
	}
	s.bumpVersion(ctx, sellerID, "categories")
}

// InvalidateAttributeLists retires attribute lists after attribute or
// link/unlink writes. Call AFTER DB commit. sellerID, when non-zero, also
// deletes that seller's per-id attribute entry; the global version bump
// retires every seller's list key.
func (s *CategoryCache) InvalidateAttributeLists(ctx context.Context, sellerID, attributeID uint) {
	if s == nil {
		return
	}
	if s.cache != nil && sellerID != 0 && attributeID != 0 {
		if k, err := cachekit.BuildSellerKey(sellerID, "attribute", uintToString(attributeID)); err == nil {
			_ = s.cache.Del(ctx, k)
		}
	}
	if s.durable != nil {
		_, _ = cachekit.BumpVersion(ctx, s.durable, "attributes:all:ver")
	}
	if sellerID != 0 {
		s.bumpVersion(ctx, sellerID, "categories")
	}
}
