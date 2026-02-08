package desc

import "github.com/delley/goas/internal/openapi"

func ExplodeRefs(fileRefPath string, spec *openapi.OpenAPIObject) error {
	if spec.Info.Description != nil {
		desc, err := FetchRef(fileRefPath, spec.Info.Description.Value)
		if err != nil {
			return err
		}
		spec.Info.Description.Value = desc
	}
	for i, tag := range spec.Tags {
		if tag.Description == nil {
			continue
		}
		desc, err := FetchRef(fileRefPath, tag.Description.Value)
		if err != nil {
			return err
		}
		spec.Tags[i].Description.Value = desc
	}

	return nil
}
