package features

import (
	"net/http"

	"github.com/PDOK/gokoala/engine"
	"github.com/PDOK/gokoala/ogc/features/domain"
)

var (
	collectionsBreadcrumb = []engine.Breadcrumb{
		{
			Name: "Collections",
			Path: "collections",
		},
	}
	featuresKey = engine.NewTemplateKey(templatesDir + "features.go.html")
	featureKey  = engine.NewTemplateKey(templatesDir + "feature.go.html")
)

type htmlFeatures struct {
	engine *engine.Engine
}

func newHTMLFeatures(e *engine.Engine) *htmlFeatures {
	e.ParseTemplate(featuresKey)
	e.ParseTemplate(featureKey)

	return &htmlFeatures{
		engine: e,
	}
}

// featureCollectionPage enriched FeatureCollection for HTML representation.
type featureCollectionPage struct {
	domain.FeatureCollection

	CollectionID string
	Metadata     *engine.GeoSpatialCollectionMetadata
	Cursor       domain.Cursor
	Limit        int
}

// featurePage enriched Feature for HTML representation.
type featurePage struct {
	domain.Feature

	FeatureID string
	Metadata  *engine.GeoSpatialCollectionMetadata
}

func (hf *htmlFeatures) features(w http.ResponseWriter, r *http.Request, collectionID string,
	cursor domain.Cursor, limit int, fc *domain.FeatureCollection) {

	collectionMetadata := collectionsMetadata[collectionID]

	breadcrumbs := collectionsBreadcrumb
	breadcrumbs = append(breadcrumbs, []engine.Breadcrumb{
		{
			Name: getCollectionTitle(collectionID, collectionMetadata),
			Path: "collections/" + collectionID,
		},
		{
			Name: "Items",
			Path: "collections/" + collectionID + "/items",
		},
	}...)

	pageContent := &featureCollectionPage{
		*fc,
		collectionID,
		collectionMetadata,
		cursor,
		limit,
	}

	lang := hf.engine.CN.NegotiateLanguage(w, r)
	hf.engine.RenderAndServePage(w, r, engine.ExpandTemplateKey(featuresKey, lang), pageContent, breadcrumbs)
}

func (hf *htmlFeatures) feature(w http.ResponseWriter, r *http.Request, collectionID string, feat *domain.Feature) {

	collectionMetadata := collectionsMetadata[collectionID]

	breadcrumbs := collectionsBreadcrumb
	breadcrumbs = append(breadcrumbs, []engine.Breadcrumb{
		{
			Name: getCollectionTitle(collectionID, collectionMetadata),
			Path: "collections/" + collectionID,
		},
		{
			Name: "Items",
			Path: "collections/" + collectionID + "/items",
		},
		{
			Name: feat.ID,
			Path: "collections/" + collectionID + "/items/" + feat.ID,
		},
	}...)

	pageContent := &featurePage{
		*feat,
		feat.ID,
		collectionMetadata,
	}

	lang := hf.engine.CN.NegotiateLanguage(w, r)
	hf.engine.RenderAndServePage(w, r, engine.ExpandTemplateKey(featureKey, lang), pageContent, breadcrumbs)
}

func getCollectionTitle(collectionID string, metadata *engine.GeoSpatialCollectionMetadata) string {
	title := collectionID
	if metadata != nil && metadata.Title != nil {
		title = *metadata.Title
	}
	return title
}