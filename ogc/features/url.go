package features

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"net/url"
	"slices"
	"sort"
	"strconv"

	"github.com/PDOK/gokoala/ogc/features/domain"
)

var (
	checksumExcludedParams = []string{"f", "cursor"} // don't include these in checksum
)

type URL interface {
	validateNoUnknownParams() error
}

// URL to a page in a collection of features
type featureCollectionURL struct {
	baseURL url.URL
	params  url.Values
}

// Calculate checksum over the query parameters that have a "filtering effect" on
// the result set such as limit, bbox, CQL filters, etc. These query params
// aren't allowed to be changed during pagination. The checksum allows for the latter
// to be verified
func (fc featureCollectionURL) checksum() []byte {
	var valuesToHash bytes.Buffer
	sortedQueryParams := make([]string, 0, len(fc.params))
	for k := range fc.params {
		sortedQueryParams = append(sortedQueryParams, k)
	}
	sort.Strings(sortedQueryParams) // sort keys
OUTER:
	for _, k := range sortedQueryParams {
		for _, skip := range checksumExcludedParams {
			if k == skip {
				continue OUTER
			}
		}
		paramValues := fc.params[k]
		if paramValues != nil {
			slices.Sort(paramValues) // sort values belonging to key
		}
		for _, s := range paramValues {
			valuesToHash.WriteString(s)
		}
	}

	bytesToHash := valuesToHash.Bytes()
	if len(bytesToHash) > 0 {
		hasher := fnv.New32a() // fast non-cryptographic hash
		hasher.Write(bytesToHash)
		return hasher.Sum(nil)
	}
	return []byte{}
}

func (fc featureCollectionURL) toSelfURL(collectionID string, format string) string {
	newParams := url.Values{}
	newParams.Set("f", format)

	result := fc.baseURL.JoinPath("collections", collectionID, "items")
	result.RawQuery = newParams.Encode()
	return result.String()
}

func (fc featureCollectionURL) toPrevNextURL(collectionID string, cursor domain.EncodedCursor, format string) string {
	copyParams := clone(fc.params)
	copyParams.Set("f", format)
	copyParams.Set("cursor", cursor.String())

	result := fc.baseURL.JoinPath("collections", collectionID, "items")
	result.RawQuery = copyParams.Encode()
	return result.String()
}

// implements req 7.6 (https://docs.ogc.org/is/17-069r4/17-069r4.html#query_parameters)
func (fc featureCollectionURL) validateNoUnknownParams() error {
	copyParams := clone(fc.params)
	copyParams.Del("f")
	copyParams.Del("limit")
	copyParams.Del("cursor")
	copyParams.Del("datetime")
	copyParams.Del("crs")
	copyParams.Del("bbox")
	copyParams.Del("bbox-crs")
	copyParams.Del("filter")
	copyParams.Del("filter-crs")
	if len(copyParams) > 0 {
		return fmt.Errorf("unknown query parameter(s) found: %v", copyParams.Encode())
	}
	return nil
}

// URL to a specific Feature
type featureURL struct {
	baseURL url.URL
	params  url.Values
}

func (f featureURL) toSelfURL(collectionID string, featureID int64, format string) string {
	newParams := url.Values{}
	newParams.Set("f", format)

	result := f.baseURL.JoinPath("collections", collectionID, "items", strconv.FormatInt(featureID, 10))
	result.RawQuery = newParams.Encode()
	return result.String()
}

func (f featureURL) toCollectionURL(collectionID string, format string) string {
	newParams := url.Values{}
	newParams.Set("f", format)

	result := f.baseURL.JoinPath("collections", collectionID)
	result.RawQuery = newParams.Encode()
	return result.String()
}

// implements req 7.6 (https://docs.ogc.org/is/17-069r4/17-069r4.html#query_parameters)
func (f featureURL) validateNoUnknownParams() error {
	copyParams := clone(f.params)
	copyParams.Del("f")
	copyParams.Del("crs")
	if len(copyParams) > 0 {
		return fmt.Errorf("unknown query parameter(s) found: %v", copyParams.Encode())
	}
	return nil
}

func clone(params url.Values) url.Values {
	copyParams := url.Values{}
	for k, v := range params {
		copyParams[k] = v
	}
	return copyParams
}
