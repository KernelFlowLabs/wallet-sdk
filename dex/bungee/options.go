package bungee

import "github.com/kernelflowlabs/wallet-sdk/dex"

var RateLimitPublic = dex.RateLimit{QPS: 5, Burst: 5}

type clientConfig struct {
	rateLimit   dex.RateLimit
	apiKey      string
	affiliateId string
}

type ClientOption func(*clientConfig)

func WithRateLimit(r dex.RateLimit) ClientOption {
	return func(c *clientConfig) { c.rateLimit = r }
}

func WithApiKey(key string) ClientOption {
	return func(c *clientConfig) { c.apiKey = key }
}

func WithAffiliateId(id string) ClientOption {
	return func(c *clientConfig) { c.affiliateId = id }
}
