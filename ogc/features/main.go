package features

import (
	"bytes"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"

	"github.com/PDOK/gokoala/engine"
	"github.com/PDOK/gokoala/ogc/common/geospatial"
	"github.com/PDOK/gokoala/ogc/features/datasources"
	"github.com/PDOK/gokoala/ogc/features/datasources/geopackage"
	"github.com/PDOK/gokoala/ogc/features/datasources/postgis"
	"github.com/PDOK/gokoala/ogc/features/domain"
	"github.com/go-chi/chi/v5"
)

const (
	templatesDir = "ogc/features/templates/"
	defaultLimit = 10
)

var (
	collectionsMetadata map[string]*engine.GeoSpatialCollectionMetadata
)

type Features struct {
	engine     *engine.Engine
	datasource datasources.Datasource

	html *htmlFeatures
	json *jsonFeatures
}

func NewFeatures(e *engine.Engine, router *chi.Mux) *Features {
	var datasource datasources.Datasource
	if e.Config.OgcAPI.Features.Datasource.GeoPackage != nil {
		datasource = geopackage.NewGeoPackage(
			e.Config.OgcAPI.Features.Collections,
			*e.Config.OgcAPI.Features.Datasource.GeoPackage)
	} else if e.Config.OgcAPI.Features.Datasource.PostGIS != nil {
		datasource = postgis.NewPostGIS()
	}
	e.RegisterShutdownHook(datasource.Close)

	f := &Features{
		engine:     e,
		datasource: datasource,
		html:       newHTMLFeatures(e),
		json:       newJSONFeatures(e),
	}
	collectionsMetadata = f.cacheCollectionsMetadata()

	router.Get(geospatial.CollectionsPath+"/{collectionId}/items", f.CollectionContent())
	router.Get(geospatial.CollectionsPath+"/{collectionId}/items/{featureId}", f.Feature())
	return f
}

// CollectionContent serve a FeatureCollection with the given collectionId
func (f *Features) CollectionContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collectionID := chi.URLParam(r, "collectionId")
		encodedCursor := domain.EncodedCursor(r.URL.Query().Get("cursor"))
		limit, err := getLimit(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = f.validateNoUnknownFeatureCollectionQueryParams(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, ok := collectionsMetadata[collectionID]; !ok {
			http.NotFound(w, r)
			return
		}

		filtersHash := f.hashQueryParams(r.URL.Query(), []string{"f", "cursor"})
		fc, newCursor, err := f.datasource.GetFeatures(r.Context(), collectionID, datasources.FeatureOptions{
			Cursor: encodedCursor.Decode(filtersHash),
			Limit:  limit,
			// TODO set bbox, bbox-crs, etc
		})
		if err != nil {
			// log error, but sent generic message to client to prevent possible information leakage from datasource
			msg := fmt.Sprintf("failed to retrieve feature collection %s", collectionID)
			log.Printf("%s, error: %v\n", msg, err)
			http.Error(w, msg, http.StatusInternalServerError)
		}
		if fc == nil {
			http.NotFound(w, r)
			return
		}

		switch f.engine.CN.NegotiateFormat(r) {
		case engine.FormatHTML:
			f.html.features(w, r, collectionID, newCursor, limit, fc)
		case engine.FormatJSON:
			f.json.featuresAsGeoJSON(w, collectionID, newCursor, limit, fc)
		case engine.FormatJSONFG:
			f.json.featuresAsJSONFG()
		default:
			http.NotFound(w, r)
		}
	}
}

func (f *Features) hashQueryParams(queryParams url.Values, exceptParams []string) []byte {
	initialSize := len(queryParams)
	var valuesToHash bytes.Buffer
	sortedQueryParams := make([]string, 0, initialSize)
	for k := range queryParams {
		sortedQueryParams = append(sortedQueryParams, k)
	}
	sort.Strings(sortedQueryParams) // sort keys
OUTER:
	for _, k := range sortedQueryParams {
		for _, except := range exceptParams {
			if k == except {
				continue OUTER
			}
		}
		paramValues := queryParams[k]
		if paramValues != nil {
			slices.Sort(paramValues) // sort values belonging to key
		}
		for _, s := range paramValues {
			valuesToHash.WriteString(s)
		}
	}
	hasher := fnv.New32a()
	hasher.Write(valuesToHash.Bytes())
	return hasher.Sum(nil)
}

// Feature serves a single Feature
func (f *Features) Feature() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collectionID := chi.URLParam(r, "collectionId")
		featureID, err := strconv.Atoi(chi.URLParam(r, "featureId"))
		if err != nil {
			http.Error(w, "feature ID must be a number", http.StatusBadRequest)
			return
		}
		if err = f.validateNoUnknownFeatureQueryParams(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		feat, err := f.datasource.GetFeature(r.Context(), collectionID, int64(featureID))
		if err != nil {
			// log error, but sent generic message to client to prevent possible information leakage from datasource
			msg := fmt.Sprintf("failed to retrieve feature %d in collection %s", featureID, collectionID)
			log.Printf("%s, error: %v\n", msg, err)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		if feat == nil {
			http.NotFound(w, r)
			return
		}

		switch f.engine.CN.NegotiateFormat(r) {
		case engine.FormatHTML:
			f.html.feature(w, r, collectionID, feat)
		case engine.FormatJSON:
			f.json.featureAsGeoJSON(w, collectionID, feat)
		case engine.FormatJSONFG:
			f.json.featureAsJSONFG()
		default:
			http.NotFound(w, r)
		}
	}
}

func (f *Features) cacheCollectionsMetadata() map[string]*engine.GeoSpatialCollectionMetadata {
	result := make(map[string]*engine.GeoSpatialCollectionMetadata)
	for _, collection := range f.engine.Config.OgcAPI.Features.Collections {
		result[collection.ID] = collection.Metadata
	}
	return result
}

func getLimit(r *http.Request) (int, error) {
	limit := defaultLimit
	var err error
	if r.URL.Query().Get("limit") != "" {
		limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil {
			err = errors.New("limit query parameter must be a number")
		}
	}
	if limit < 0 {
		err = errors.New("limit can't be negative")
	}
	return limit, err
}

// validateNoUnknownFeatureCollectionQueryParams implements req 7.6 (https://docs.ogc.org/is/17-069r4/17-069r4.html#query_parameters)
func (f *Features) validateNoUnknownFeatureCollectionQueryParams(r *http.Request) error {
	copyQueryString := r.URL.Query()
	copyQueryString.Del("f")
	copyQueryString.Del("limit")
	copyQueryString.Del("cursor")
	copyQueryString.Del("datetime")
	copyQueryString.Del("crs")
	copyQueryString.Del("bbox")
	copyQueryString.Del("bbox-crs")
	copyQueryString.Del("filter")
	copyQueryString.Del("filter-crs")
	if len(copyQueryString) > 0 {
		return fmt.Errorf("unknown query parameter(s) found: %v", copyQueryString.Encode())
	}
	return nil
}

// validateNoUnknownFeatureQueryParams implements req 7.6 (https://docs.ogc.org/is/17-069r4/17-069r4.html#query_parameters)
func (f *Features) validateNoUnknownFeatureQueryParams(r *http.Request) error {
	copyQueryString := r.URL.Query()
	copyQueryString.Del("f")
	copyQueryString.Del("crs")
	if len(copyQueryString) > 0 {
		return fmt.Errorf("unknown query parameter(s) found: %v", copyQueryString.Encode())
	}
	return nil
}
