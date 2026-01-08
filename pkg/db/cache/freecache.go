package cache

import (
	"fmt"

	"github.com/coocood/freecache"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/sync/singleflight"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type FC struct {
	cache *freecache.Cache
	sf    *singleflight.Group
}

func NewFreeCache(sizeMB int) *FC {
	size := sizeMB * 1024 * 1024
	return &FC{
		cache: freecache.NewCache(size),
		sf:    &singleflight.Group{},
	}
}

func (f *FC) SetCache(k string, v interface{}, ttlSeconds int) error {
	vb, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json-iterator marshal error: %w", err)
	}
	return f.cache.Set([]byte(k), vb, ttlSeconds)
}

func (f *FC) GetCache(k string, v interface{}) error {
	vb, err := f.cache.Get([]byte(k))
	if err != nil {
		return err
	}
	return json.Unmarshal(vb, v)
}

func (f *FC) GetOrLoadCache(k string, v interface{}, ttlSeconds int, loader func() (interface{}, error)) error {
	if err := f.GetCache(k, v); err == nil {
		return nil
	}

	val, err, _ := f.sf.Do(k, func() (interface{}, error) {
		if err := f.GetCache(k, v); err == nil {
			return v, nil
		}

		data, err := loader()
		if err != nil {
			return nil, err
		}

		if err := f.SetCache(k, data, ttlSeconds); err != nil {
			return nil, err
		}

		return data, nil
	})

	if err != nil {
		return err
	}

	if val != v {
		vb, _ := json.Marshal(val)
		return json.Unmarshal(vb, v)
	}

	return nil
}

func (f *FC) DelCache(k string) bool {
	return f.cache.Del([]byte(k))
}

func (f *FC) GetHitRate() float64 {
	return f.cache.HitRate()
}
