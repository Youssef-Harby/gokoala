package domain

import (
	"fmt"
	"sort"
	"time"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/geom/encoding/geojson"
	"github.com/jmoiron/sqlx"
)

// featureCollectionType allows the GeoJSON type to be automatically set during json marshalling
type featureCollectionType struct{}

func (fc *featureCollectionType) MarshalJSON() ([]byte, error) {
	return []byte(`"FeatureCollection"`), nil
}

func (fc *featureCollectionType) UnmarshalJSON([]byte) error { return nil }

// FeatureCollection is a GeoJSON FeatureCollection with extras such as links
type FeatureCollection struct {
	Links []Link `json:"links,omitempty"`

	NumberReturned int                   `json:"numberReturned"`
	Type           featureCollectionType `json:"type"`
	Features       []*Feature            `json:"features"`
}

// Feature is a GeoJSON Feature with extras such as links
type Feature struct {
	// we overwrite ID since we want to make it a required attribute. We also expect feature ids to be
	// auto-incrementing integers (which is the default in geopackages) since we use it for cursor-based pagination.
	ID    int64  `json:"id"`
	Links []Link `json:"links,omitempty"`

	geojson.Feature
}

// Link according to RFC 8288, https://datatracker.ietf.org/doc/html/rfc8288
type Link struct {
	Length    int64  `json:"length,omitempty"`
	Rel       string `json:"rel"`
	Title     string `json:"title,omitempty"`
	Type      string `json:"type,omitempty"`
	Href      string `json:"href"`
	Hreflang  string `json:"hreflang,omitempty"`
	Templated bool   `json:"templated,omitempty"`
}

// MapRowsToFeatures datasource agnostic mapper from SQL rows/result set to Features domain model
func MapRowsToFeatures(rows *sqlx.Rows, fidColumn string, geomColumn string,
	geomMapper func([]byte) (geom.Geometry, error)) ([]*Feature, error) {

	result := make([]*Feature, 0)
	columns, err := rows.Columns()
	if err != nil {
		return result, err
	}

	for rows.Next() {
		var values []interface{}
		if values, err = rows.SliceScan(); err != nil {
			return result, err
		}
		feature := &Feature{Feature: geojson.Feature{Properties: make(map[string]interface{})}}

		if err = mapColumnsToFeature(feature, columns, values, fidColumn, geomColumn, geomMapper); err != nil {
			return result, err
		}
		result = append(result, feature)
	}

	// sort by ascending ID, we need sorting here since 'previous' navigation causes an inverted result set
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

//nolint:cyclop
func mapColumnsToFeature(feature *Feature, columns []string, values []interface{}, fidColumn string,
	geomColumn string, geomMapper func([]byte) (geom.Geometry, error)) error {

	for i, columnName := range columns {
		columnValue := values[i]
		if columnValue == nil {
			continue
		}

		switch columnName {
		case fidColumn:
			feature.ID = columnValue.(int64)

		case geomColumn:
			rawGeom, ok := columnValue.([]byte)
			if !ok {
				return fmt.Errorf("failed to read geometry from %s column in datasource", geomColumn)
			}
			mappedGeom, err := geomMapper(rawGeom)
			if err != nil {
				return fmt.Errorf("failed to map/decode geometry from datasource, error: %w", err)
			}
			feature.Geometry = geojson.Geometry{Geometry: mappedGeom}

		case "minx", "miny", "maxx", "maxy", "min_zoom", "max_zoom":
			// Skip these columns used for bounding box and zoom filtering
			continue

		default:
			// Grab any non-nil, non-id, non-bounding box, & non-geometry column as a tag
			switch v := columnValue.(type) {
			case []uint8:
				asBytes := make([]byte, len(v))
				copy(asBytes, v)
				feature.Properties[columnName] = string(asBytes)
			case int64:
				feature.Properties[columnName] = v
			case float64:
				feature.Properties[columnName] = v
			case time.Time:
				feature.Properties[columnName] = v
			case string:
				feature.Properties[columnName] = v
			case bool:
				feature.Properties[columnName] = v
			default:
				return fmt.Errorf("unexpected type for sqlite column data: %v: %T", columns[i], v)
			}
		}
	}
	return nil
}
