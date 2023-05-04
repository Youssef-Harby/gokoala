package maps

import (
	"gokoala/engine"
	"gokoala/ogc/common/geospatial"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Maps !!! Placeholder implementation, for future reference !!!
type Maps struct {
	engine *engine.Engine
}

// NewMaps !!! Placeholder implementation, for future reference !!!
func NewMaps(e *engine.Engine, router *chi.Mux) *Maps {
	maps := &Maps{
		engine: e,
	}

	router.Get(geospatial.CollectionsPath+"/{collectionId}/map", maps.CollectionContent())
	router.Get(geospatial.CollectionsPath+"/{collectionId}/map/tiles", maps.CollectionContent())
	return maps
}

func (t *Maps) CollectionContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collectionID := chi.URLParam(r, "collectionId")

		// TODO: not implemented yet
		log.Printf("TODO: return maps for collection %s", collectionID)
	}
}
