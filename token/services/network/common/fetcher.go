/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

type Fetcher interface {
	FetchPublicParameters(namespace string) ([]byte, error)
}

type publicParamsFetcher struct {
	fetcher   Fetcher
	namespace string
}

func NewPublicParamsFetcher(fetcher Fetcher, namespace string) *publicParamsFetcher {
	return &publicParamsFetcher{
		fetcher:   fetcher,
		namespace: namespace,
	}
}

func (c *publicParamsFetcher) Fetch() ([]byte, error) {
	return c.fetcher.FetchPublicParameters(c.namespace)
}
