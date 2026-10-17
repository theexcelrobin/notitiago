package notitiago

// Ptr returns a pointer to v, for the optional fields that need to tell zero
// apart from unset, such as Content.TTL or UpdateRequest.Title.
func Ptr[T any](v T) *T { return &v }
