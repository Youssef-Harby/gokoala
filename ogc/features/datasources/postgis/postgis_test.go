package postgis

import (
	"context"
	"testing"

	"github.com/PDOK/gokoala/ogc/features/datasources"
	"github.com/stretchr/testify/assert"
)

// PostGIS !!! Placeholder implementation, for future reference !!!
func TestPostGIS(t *testing.T) {
	pg := PostGIS{}

	t.Run("GetFeatureIDs", func(t *testing.T) {
		ids, cursors, err := pg.GetFeatureIDs(context.Background(), "", datasources.FeaturesCriteria{})
		assert.NoError(t, err)
		assert.Empty(t, ids)
		assert.NotNil(t, cursors)
	})

	t.Run("GetFeaturesByID", func(t *testing.T) {
		fc, err := pg.GetFeaturesByID(context.Background(), "", nil)
		assert.NoError(t, err)
		assert.NotNil(t, fc)
	})

	t.Run("GetFeatures", func(t *testing.T) {
		fc, cursors, err := pg.GetFeatures(context.Background(), "", datasources.FeaturesCriteria{})
		assert.NoError(t, err)
		assert.Nil(t, fc)
		assert.NotNil(t, cursors)
	})

	t.Run("GetFeature", func(t *testing.T) {
		f, err := pg.GetFeature(context.Background(), "", 0)
		assert.NoError(t, err)
		assert.Nil(t, f)
	})

	t.Run("GetFeatureTableMetadata", func(t *testing.T) {
		metadata, err := pg.GetFeatureTableMetadata("")
		assert.NoError(t, err)
		assert.Nil(t, metadata)
	})
}
