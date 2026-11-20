package permission

import "context"

func CategoryResource(category string) Resource {
	return Resource{Type: Category, Value: category}
}

func LibraryResource(reference string) Resource {
	return Resource{Type: Library, Value: reference}
}

func FileResource(reference, filename string) Resource {
	return Resource{Type: File, Value: reference + "/" + filename}
}

// Resources orders explicit file permissions before library and category defaults.
func Resources(category, identifie, filename string) []Resource {
	reference := category + "/" + identifie
	resources := []Resource{LibraryResource(reference), CategoryResource(category)}
	if filename != "" {
		resources = append([]Resource{FileResource(reference, filename)}, resources...)
	}
	return resources
}

func Allowed(ctx context.Context, service Service, token string, resources []Resource, action string) (bool, error) {
	decision, err := service.Check(ctx, token, resources, action)
	return decision.Allowed, err
}

func Require(ctx context.Context, service Service, token string, resources []Resource, action string) error {
	allowed, err := Allowed(ctx, service, token, resources, action)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrDenied
	}
	return nil
}

// FileActions checks in bounded batches and returns only capabilities, never external grant records.
func FileActions(ctx context.Context, service Service, token, category, identifie string, paths, actions []string) (map[string]map[string]bool, error) {
	result := make(map[string]map[string]bool, len(paths))
	for start := 0; start < len(paths); start += 100 {
		end := start + 100
		if end > len(paths) {
			end = len(paths)
		}
		requests := make([]Request, 0, (end-start)*len(actions))
		for _, filename := range paths[start:end] {
			for _, action := range actions {
				requests = append(requests, Request{Resources: Resources(category, identifie, filename), Action: action})
			}
		}
		decisions, err := service.Batch(ctx, token, requests)
		if err != nil {
			return nil, err
		}
		if len(decisions) != len(requests) {
			return nil, ErrDenied
		}
		index := 0
		for _, filename := range paths[start:end] {
			result[filename] = map[string]bool{}
			for _, action := range actions {
				result[filename][action] = decisions[index].Allowed
				index++
			}
		}
	}
	return result, nil
}
