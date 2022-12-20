package utils

type HandlerFunc func(interface{}) bool

// Filter a slice with custom filter function
func Filter(items []map[string]interface{}, FilterFunc HandlerFunc) []map[string]interface{} {
    results := make([]map[string]interface{}, 0)

    for _, item := range items {
        if FilterFunc(item) {
            results = append(results, item)
        }
    }

    return results
}

// Filter all not nil items in a slice
func FilterNotNil(items []map[string]interface{}) []map[string]interface{} {
    return Filter(items, func(item interface{}) bool {
        return item != nil
    })
}