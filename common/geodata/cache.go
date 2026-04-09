package geodata

import (
	"os"
	"strings"
	"sync"

	v2router "github.com/v2fly/v2ray-core/v4/app/router"
	"google.golang.org/protobuf/proto"

	"github.com/voidluo/trojan-go/common"
	"github.com/voidluo/trojan-go/log"
)

type GenericCache[T any] struct {
	sync.RWMutex
	data map[string]*T
}

func (c *GenericCache[T]) Get(key string) *T {
	c.RLock()
	defer c.RUnlock()
	if c.data == nil {
		return nil
	}
	return c.data[key]
}

func (c *GenericCache[T]) Set(key string, value *T) {
	c.Lock()
	defer c.Unlock()
	if c.data == nil {
		c.data = make(map[string]*T)
	}
	c.data[key] = value
}

type geoipCache struct {
	GenericCache[v2router.GeoIP]
}

func (g *geoipCache) Unmarshal(filename, code string) (*v2router.GeoIP, error) {
	asset := common.GetAssetLocation(filename)
	idx := strings.ToLower(asset + ":" + code)
	if entry := g.Get(idx); entry != nil {
		log.Debugf("geoip cache HIT: %s -> %s", code, idx)
		return entry, nil
	}

	geoipBytes, err := Decode(asset, code)
	switch err {
	case nil:
		var geoip v2router.GeoIP
		if err := proto.Unmarshal(geoipBytes, &geoip); err != nil {
			return nil, err
		}
		g.Set(idx, &geoip)
		return &geoip, nil

	case ErrCodeNotFound:
		return nil, common.NewError("country code " + code + " not found in " + filename)

	case ErrFailedToReadBytes, ErrFailedToReadExpectedLenBytes,
		ErrInvalidGeodataFile, ErrInvalidGeodataVarintLength:
		log.Warnf("failed to decode geoip file: %s, fallback to the original ReadFile method", filename)
		geoipBytes, err := os.ReadFile(asset)
		if err != nil {
			return nil, err
		}
		var geoipList v2router.GeoIPList
		if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
			return nil, err
		}
		for _, geoip := range geoipList.GetEntry() {
			if strings.EqualFold(code, geoip.GetCountryCode()) {
				g.Set(idx, geoip)
				return geoip, nil
			}
		}

	default:
		return nil, err
	}

	return nil, common.NewError("country code " + code + " not found in " + filename)
}

type geositeCache struct {
	GenericCache[v2router.GeoSite]
}

func (g *geositeCache) Unmarshal(filename, code string) (*v2router.GeoSite, error) {
	asset := common.GetAssetLocation(filename)
	idx := strings.ToLower(asset + ":" + code)
	if entry := g.Get(idx); entry != nil {
		log.Debugf("geosite cache HIT: %s -> %s", code, idx)
		return entry, nil
	}

	geositeBytes, err := Decode(asset, code)
	switch err {
	case nil:
		var geosite v2router.GeoSite
		if err := proto.Unmarshal(geositeBytes, &geosite); err != nil {
			return nil, err
		}
		g.Set(idx, &geosite)
		return &geosite, nil

	case ErrCodeNotFound:
		return nil, common.NewError("list " + code + " not found in " + filename)

	case ErrFailedToReadBytes, ErrFailedToReadExpectedLenBytes,
		ErrInvalidGeodataFile, ErrInvalidGeodataVarintLength:
		log.Warnf("failed to decode geoip file: %s, fallback to the original ReadFile method", filename)
		geositeBytes, err := os.ReadFile(asset)
		if err != nil {
			return nil, err
		}
		var geositeList v2router.GeoSiteList
		if err := proto.Unmarshal(geositeBytes, &geositeList); err != nil {
			return nil, err
		}
		for _, geosite := range geositeList.GetEntry() {
			if strings.EqualFold(code, geosite.GetCountryCode()) {
				g.Set(idx, geosite)
				return geosite, nil
			}
		}

	default:
		return nil, err
	}

	return nil, common.NewError("list " + code + " not found in " + filename)
}
